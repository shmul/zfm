package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Play executes a Recipe and plays the result via ffplay.
// Volume stats are computed concurrently and printed as soon as they are ready.
func Play(r Recipe, label ...string) error {
	playPath := r.InputPath

	if !r.Identical {
		ext := filepath.Ext(r.InputPath)
		tmp, err := os.CreateTemp("", "zfm-*"+ext)
		if err != nil {
			return err
		}
		tmp.Close()
		defer os.Remove(tmp.Name())

		if err := Execute(r, tmp.Name()); err != nil {
			return err
		}
		playPath = tmp.Name()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	prefix := ""
	if len(label) > 0 {
		prefix = label[0] + " "
	}
	go func() {
		defer wg.Done()
		vol, err := VolumeStats(playPath)
		if err == nil {
			fmt.Printf("  %s%.1f / %.1f dBFS\n", prefix, vol.Mean, vol.Peak)
		}
	}()

	err := ffplay(playPath)
	wg.Wait()
	return err
}

// PlayFile plays a file directly (no cropping) via ffplay.
func PlayFile(path string) error {
	return ffplay(path)
}

// StartPlay starts playback of a Recipe asynchronously and returns immediately.
// All preparation (Execute) and playback run in a background goroutine.
// The returned done channel closes when playback ends. Call stop() to cancel.
func StartPlay(r Recipe) (stop func(), done <-chan struct{}) {
	ch := make(chan struct{})
	stopCh := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(ch)

		playPath, tmpPath, err := preparePlayPath(r)
		if tmpPath != "" {
			defer os.Remove(tmpPath) //nolint:errcheck
		}
		if err != nil {
			return
		}

		// Bail out early if stop was already requested during Execute.
		select {
		case <-stopCh:
			return
		default:
		}

		abs, err := filepath.Abs(playPath)
		if err != nil {
			return
		}

		cmd := exec.Command("ffplay", "-nodisp", "-autoexit", abs)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return
		}

		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()

		select {
		case <-stopCh:
			cmd.Process.Kill() //nolint:errcheck
			<-waitCh
		case <-waitCh:
		}
	}()

	stop = func() { once.Do(func() { close(stopCh) }) }
	return stop, ch
}

// StartPlayAt starts playback of a file directly using ffplay's built-in seeking,
// bypassing ffmpeg preprocessing. Returns immediately; done closes when playback ends.
func StartPlayAt(path string, ss, to float64) (stop func(), done <-chan struct{}) {
	ch := make(chan struct{})
	stopCh := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(ch)

		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}

		args := []string{"-nodisp", "-autoexit", "-ss", fmt.Sprintf("%.3f", ss)}
		if to > ss {
			args = append(args, "-t", fmt.Sprintf("%.3f", to-ss))
		}
		args = append(args, abs)

		cmd := exec.Command("ffplay", args...)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return
		}

		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()

		select {
		case <-stopCh:
			cmd.Process.Kill() //nolint:errcheck
			<-waitCh
		case <-waitCh:
		}
	}()

	stop = func() { once.Do(func() { close(stopCh) }) }
	return stop, ch
}

func preparePlayPath(r Recipe) (playPath, tmpPath string, err error) {
	if r.Identical {
		return r.InputPath, "", nil
	}
	ext := filepath.Ext(r.InputPath)
	tmp, e := os.CreateTemp("", "zfm-*"+ext)
	if e != nil {
		return "", "", e
	}
	tmp.Close()
	tmpPath = tmp.Name()
	if e := Execute(r, tmpPath); e != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return "", "", e
	}
	return tmpPath, tmpPath, nil
}

func ffplay(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	cmd, err := ProcsCmdStr("ffplay", []string{"-nodisp", "-autoexit", abs})
	if err != nil {
		return err
	}
	return newCmd(cmd).Run()
}
