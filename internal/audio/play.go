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
