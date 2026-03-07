package audio

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PrepareParams describes the input parameters for Prepare.
type PrepareParams struct {
	Path      string
	Start     string  // [hh:]mm:ss
	End       string  // [hh:]mm:ss
	Head      float64 // seconds offset from start
	Tail      float64 // seconds to trim from end
	FadeIn    float64 // seconds
	FadeOut   float64 // seconds
	FadeCurve string  // afade curve type (e.g. "qsin", "tri"); defaults to "qsin"
}

// Recipe describes how to produce a cropped audio file from a source.
type Recipe struct {
	InputPath string
	SS        float64 // start offset in seconds
	To        float64 // end offset in seconds (0 = full duration)
	FadeIn    float64 // seconds
	FadeOut   float64 // seconds
	FadeCurve string  // afade curve type (e.g. "qsin", "tri"); defaults to "qsin"
	Duration  float64 // total source duration
	// Identical is true when no audio operations are applied (symlink candidate).
	Identical bool
}

// ParseTime parses a time string into seconds.
// Accepts Go duration syntax (e.g. "46m57s", "1h32m5s") or [hh:]mm:ss notation.
// Returns 0 and no error on empty input.
func ParseTime(hms string) (float64, error) {
	if hms == "" {
		return 0, nil
	}
	if strings.ContainsAny(hms, "hms") {
		d, err := time.ParseDuration(hms)
		if err != nil {
			return 0, fmt.Errorf("invalid time format: %q", hms)
		}
		return d.Seconds(), nil
	}
	parts := strings.Split(hms, ":")
	var goFmt string
	switch len(parts) {
	case 2:
		goFmt = parts[0] + "m" + parts[1] + "s"
	case 3:
		goFmt = parts[0] + "h" + parts[1] + "m" + parts[2] + "s"
	default:
		return 0, fmt.Errorf("invalid time format: %q", hms)
	}
	d, err := time.ParseDuration(goFmt)
	if err != nil {
		return 0, fmt.Errorf("invalid time format: %q", hms)
	}
	return d.Seconds(), nil
}

// Prepare computes a Recipe from crop parameters.
// The ProbeInfo used during calculation is returned for callers that need it.
func Prepare(p PrepareParams) (Recipe, ProbeInfo, error) {
	if p.FadeCurve == "" {
		p.FadeCurve = "qsin"
	}
	info, err := Probe(p.Path)
	if err != nil {
		return Recipe{}, ProbeInfo{}, err
	}

	ss, err := startOffset(p.Start, p.Head)
	if err != nil {
		return Recipe{}, info, err
	}

	to, err := endOffset(p.End, p.Tail, info.Duration)
	if err != nil {
		return Recipe{}, info, err
	}

	identical := ss == 0 && to == info.Duration && p.FadeIn == 0 && p.FadeOut == 0
	return Recipe{
		InputPath: p.Path,
		SS:        ss,
		To:        to,
		FadeIn:    p.FadeIn,
		FadeOut:   p.FadeOut,
		FadeCurve: p.FadeCurve,
		Duration:  info.Duration,
		Identical: identical,
	}, info, nil
}

func startOffset(hms string, head float64) (float64, error) {
	if hms != "" {
		return ParseTime(hms)
	}
	return head, nil
}

func endOffset(hms string, tail, totalDuration float64) (float64, error) {
	if hms != "" {
		return ParseTime(hms)
	}
	if tail != 0 {
		return math.Max(0, totalDuration-math.Abs(tail)), nil
	}
	return totalDuration, nil
}

// Execute writes the Recipe output to dest.
// Behaviour:
//   - Identical  → symlink
//   - No fades   → ffmpeg -c copy (lossless)
//   - Fades      → ffmpeg re-encode with afade filter
func Execute(r Recipe, dest string) error {
	if r.Identical {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(r.InputPath, dest)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	ss := fmt.Sprintf("%.3f", r.SS)
	to := fmt.Sprintf("%.3f", r.To)

	var ffArgs []string
	if r.FadeIn == 0 && r.FadeOut == 0 {
		ffArgs = []string{"-ss", ss, "-to", to, "-i", r.InputPath, "-c", "copy", "-y", dest}
	} else {
		// asetpts=PTS-STARTPTS normalises timestamps to 0 after the seek so that
		// afade positions are relative to the segment start, not the original file.
		segDur := r.To - r.SS
		ffArgs = []string{"-ss", ss, "-to", to, "-i", r.InputPath, "-af", buildFadeFilter(segDur, r.FadeIn, r.FadeOut, r.FadeCurve), "-y", dest}
	}
	cmd, err := ProcsCmdStr("ffmpeg", ffArgs)
	if err != nil {
		return err
	}
	return newCmd(cmd).Run()
}

func buildFadeFilter(segDur, fadeIn, fadeOut float64, curve string) string {
	parts := []string{"asetpts=PTS-STARTPTS"}
	if fadeIn > 0 {
		parts = append(parts, fmt.Sprintf("afade=t=in:st=0:d=%.3f:curve=%s", fadeIn, curve))
	}
	if fadeOut > 0 {
		parts = append(parts, fmt.Sprintf("afade=t=out:st=%.3f:d=%.3f:curve=%s",
			math.Max(0, segDur-fadeOut), fadeOut, curve))
	}
	return strings.Join(parts, ",")
}
