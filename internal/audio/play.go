package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Play executes a Recipe and plays the result via ffplay.
// Volume stats are computed concurrently and printed as soon as they are ready.
func Play(r Recipe, label ...string) error {
	playPath, tmpPath, err := preparePlayPath(r)
	if tmpPath != "" {
		defer os.Remove(tmpPath) //nolint:errcheck
	}
	if err != nil {
		return err
	}

	prefix := ""
	if len(label) > 0 {
		prefix = label[0] + " "
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vol, err := VolumeStats(playPath)
		if err == nil {
			fmt.Printf("  %s%.1f / %.1f dBFS\n", prefix, vol.Mean, vol.Peak)
		}
	}()

	err = ffplay(playPath)
	wg.Wait()
	return err
}

// PlayFile plays a file directly (no cropping) via ffplay.
func PlayFile(path string) error {
	return ffplay(path)
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

		cmdStr, err := ProcsCmdStr("ffplay", args)
		if err != nil {
			return
		}
		p := newCmd(cmdStr)
		if err := p.Start(); err != nil {
			return
		}

		waitCh := make(chan error, 1)
		go func() { waitCh <- p.Wait() }()

		select {
		case <-stopCh:
			p.Cmds[len(p.Cmds)-1].Process.Kill() //nolint:errcheck
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
