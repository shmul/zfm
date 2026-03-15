package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// ConcatRecipes encodes and concatenates audio slices into output using a single
// ffmpeg invocation with filter_complex. No intermediate temp files are created.
func ConcatRecipes(recipes []Recipe, output string) error {
	if len(recipes) == 0 {
		return nil
	}

	args := []string{"-loglevel", "error"}
	for _, r := range recipes {
		args = append(args, "-i", r.InputPath)
	}

	var filterParts []string
	var concatInputs string
	for i, r := range recipes {
		label := fmt.Sprintf("a%d", i)
		filterParts = append(filterParts, fmt.Sprintf("[%d:a]%s[%s]", i, buildSliceFilter(r), label))
		concatInputs += fmt.Sprintf("[%s]", label)
	}
	filterParts = append(filterParts, fmt.Sprintf("%sconcat=n=%d:v=0:a=1[out]", concatInputs, len(recipes)))

	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "[out]",
		"-b:a", "320k",
		"-y", output,
	)

	cmd, err := ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return err
	}
	return newCmd(cmd).Run()
}

func buildSliceFilter(r Recipe) string {
	to := r.To
	if to == 0 {
		to = r.Duration
	}
	segDur := to - r.SS
	return fmt.Sprintf("atrim=start=%.3f:end=%.3f,", r.SS, to) +
		buildAudioFilter(segDur, r.FadeIn, r.FadeOut, r.FadeCurve, r.Volume)
}

// ConcatFiles concatenates audio files into a single MP3 at output.
func ConcatFiles(paths []string, output string) error {
	f, err := os.CreateTemp("", "zfm-concat-*.txt")
	if err != nil {
		return err
	}
	listFile := f.Name()
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

	args := []string{"-loglevel", "error", "-f", "concat", "-safe", "0", "-i", listFile, "-b:a", "320k", "-y", output}
	cmd, err := ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return err
	}
	return newCmd(cmd).Run()
}
