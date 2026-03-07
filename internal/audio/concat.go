package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ionrock/procs"
)

// FmtDuration formats a duration in seconds as [h:]mm:ss.
func FmtDuration(secs float64) string {
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// ConcatFiles concatenates audio files into a single MP3 at output.
func ConcatFiles(paths []string, output string) error {
	listFile := filepath.Join(filepath.Dir(output), "concat.txt")
	f, err := os.Create(listFile)
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove(listFile) //nolint:errcheck
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "file '%s'\n", strings.ReplaceAll(abs, "'", "'\\''")); err != nil {
			return err
		}
	}

	args := []string{"-f", "concat", "-safe", "0", "-i", listFile, "-b:a", "320k", "-y", output}
	cmd, err := ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return err
	}
	p := procs.NewProcess(cmd)
	p.ErrHandler = func(line string) string { fmt.Fprintln(os.Stderr, line); return line }
	return p.Run()
}
