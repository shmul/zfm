package mix

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/shmul/zfm/internal/audio"
)

type (
	Params struct {
		MixFile       string
		TargetDir     string
		Just          []int
		DryRun        bool
		Preview       float64
		Plot          bool
		SilenceThresh float64
	}

	sliceResult struct {
		idx      int
		artist   string
		title    string
		duration float64
		path     string
		recipe   audio.Recipe
	}
)

func Run(p Params) error {
	mf, err := Parse(p.MixFile)
	if err != nil {
		return err
	}

	if p.Plot {
		return runPlot(p, mf)
	}

	destDir := p.TargetDir
	if destDir == "" {
		destDir = filepath.Dir(p.MixFile)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	results, cleanup, err := processSlices(p, mf, destDir)
	defer cleanup()
	if err != nil {
		return err
	}

	return finalize(p, results, filepath.Join(destDir, mf.Output))
}

func processSlices(p Params, mf MixFile, destDir string) ([]sliceResult, func(), error) {
	results := make([]sliceResult, len(mf.Mix))
	var tmpPaths []string

	for i, s := range mf.Mix {
		if !shouldProcess(p.Just, i) {
			continue
		}

		r, info, err := audio.Prepare(audio.PrepareParams{
			Path:      mf.Tracks[s.Track],
			Start:     s.Start,
			End:       s.End,
			Head:      s.Head,
			Tail:      s.Tail,
			FadeIn:    s.FadeIn,
			FadeOut:   s.FadeOut,
			FadeCurve: s.FadeCurve,
		})
		if err != nil {
			return nil, noop, err
		}

		dur := r.To - r.SS
		if dur <= 0 {
			dur = r.Duration
		}

		sr := sliceResult{idx: i, artist: info.Tags["ARTIST"], title: info.Tags["TITLE"], duration: dur, recipe: r}

		if p.DryRun {
			fmt.Printf("  [dry-run] slice %d track=%s ss=%.3f to=%.3f fade_in=%.3f fade_out=%.3f identical=%v\n",
				i, s.Track, r.SS, r.To, r.FadeIn, r.FadeOut, r.Identical)
		} else if p.Preview == 0 {
			path, tmp, err := executeToTemp(i, r, destDir)
			if err != nil {
				return nil, noop, err
			}
			if tmp {
				tmpPaths = append(tmpPaths, path)
			}
			sr.path = path
		}

		results[i] = sr
	}

	cleanup := func() {
		for _, p := range tmpPaths {
			os.Remove(p) //nolint:errcheck
		}
	}
	return results, cleanup, nil
}

// executeToTemp encodes a slice to a temp file for later concatenation.
//
// Concatenation uses ffmpeg's concat demuxer (pass 2), which requires each input
// to be a complete, fully-encoded file. Slices that need cropping or fading are
// therefore pre-processed here into temp files (pass 1). Identical slices skip
// this and reuse the original file directly.
func executeToTemp(i int, r audio.Recipe, destDir string) (path string, isTmp bool, err error) {
	if r.Identical {
		return r.InputPath, false, nil
	}
	tmp, err := os.CreateTemp(destDir, fmt.Sprintf("zfm-mix-%d-*.mp3", i))
	if err != nil {
		return "", false, err
	}
	tmp.Close()
	if err := audio.Execute(r, tmp.Name()); err != nil {
		os.Remove(tmp.Name()) //nolint:errcheck
		return "", false, err
	}
	return tmp.Name(), true, nil
}

func finalize(p Params, results []sliceResult, output string) error {
	if p.Preview > 0 {
		for _, sr := range results {
			if sr.recipe.InputPath != "" {
				previewSlice(sr, p.Preview, p.SilenceThresh)
			}
		}
		return nil
	}

	if p.DryRun {
		return nil
	}

	printTracklist(results)

	if len(p.Just) > 0 {
		return nil
	}

	var paths []string
	for _, sr := range results {
		if sr.path != "" {
			paths = append(paths, sr.path)
		}
	}
	return audio.ConcatFiles(paths, output)
}

func noop() {}

func shouldProcess(just []int, i int) bool {
	return len(just) == 0 || slices.Contains(just, i)
}

func printTracklist(results []sliceResult) {
	var acc float64
	for _, sr := range results {
		if sr.path == "" {
			continue
		}
		fmt.Printf("(%s) [%s] %s\n", audio.FmtDuration(acc), audio.FmtDuration(sr.duration), trackLabel(sr))
		acc += sr.duration
	}
}

func trackLabel(sr sliceResult) string {
	if sr.title != "" {
		return sr.artist + " - " + sr.title
	}
	if sr.artist != "" {
		return sr.artist
	}
	return fmt.Sprintf("slice %d", sr.idx)
}

func previewSlice(sr sliceResult, previewSecs, silenceThresh float64) {
	fmt.Printf("\n==== %02d [%s] %s\n", sr.idx, audio.FmtDuration(sr.duration), trackLabel(sr))

	r := sr.recipe

	headR := r
	headR.To = math.Min(r.SS+previewSecs, r.To)
	headR.FadeOut = 0
	headR.Identical = false
	audio.Play(headR, "head:") //nolint:errcheck

	tailR := r
	tailR.SS = math.Max(r.SS, r.To-previewSecs)
	tailR.FadeIn = 0
	tailR.Identical = false

	preTailTo := tailR.SS
	preTailSS := math.Max(r.SS, preTailTo-previewSecs)
	if vol, err := audio.VolumeStatsRange(r.InputPath, preTailSS, preTailTo); err == nil {
		fmt.Printf("  pre-tail: %.1f / %.1f dBFS\n", vol.Mean, vol.Peak)
	}
	if profile, err := audio.VolumeProfile(r.InputPath, tailR.SS, r.To, 1.0); err == nil {
		fmt.Print(audio.PlotProfile(profile, silenceThresh, 0))
		fmt.Print(audio.FormatProfile(profile, r.To, silenceThresh))
	}

	audio.Play(tailR, "tail:") //nolint:errcheck
}
