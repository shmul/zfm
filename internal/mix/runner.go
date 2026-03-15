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
		TracksOnly    bool
		SilenceThresh float64
	}

	sliceResult struct {
		idx      int
		artist   string
		title    string
		duration float64
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

	results, err := processSlices(p, mf)
	if err != nil {
		return err
	}

	return finalize(p, results, filepath.Join(destDir, mf.Output))
}

func processSlices(p Params, mf MixFile) ([]sliceResult, error) {
	results := make([]sliceResult, len(mf.Mix))

	var acc float64
	for i, s := range mf.Mix {
		if !shouldProcess(p.Just, i) {
			continue
		}

		r, info, err := audio.Prepare(audio.PrepareParams{
			Path:         mf.Tracks[s.Track],
			Start:        s.Start,
			End:          s.End,
			Head:         s.Head,
			Tail:         s.Tail,
			FadeIn:       s.FadeIn,
			FadeOut:      s.FadeOut,
			FadeCurve:    s.FadeCurve,
			Volume:       s.Volume,
			TargetVolume: s.TargetVolume,
		})
		if err != nil {
			return nil, err
		}

		dur := r.To - r.SS
		if dur <= 0 {
			dur = r.Duration
		}

		sr := sliceResult{idx: i, artist: info.Tags["ARTIST"], title: info.Tags["TITLE"], duration: dur, recipe: r}

		if p.DryRun {
			tvStr := ""
			if s.TargetVolume != nil {
				tvStr = fmt.Sprintf(" target_volume=%.1f→%.1f", *s.TargetVolume, r.Volume)
			}
			fmt.Printf("  [dry-run] slice %d track=%s ss=%.3f to=%.3f fade_in=%.3f fade_out=%.3f volume=%.1f%s identical=%v\n",
				i, s.Track, r.SS, r.To, r.FadeIn, r.FadeOut, r.Volume, tvStr, r.Identical)
		} else if p.Preview == 0 && !p.TracksOnly {
			fmt.Printf("(%s) [%s] %s\n", audio.FmtDuration(acc), audio.FmtDuration(sr.duration), trackLabel(sr))
			acc += sr.duration
		}

		results[i] = sr
	}

	if acc > 0 {
		fmt.Printf("  total: %s\n", audio.FmtDuration(acc))
	}

	return results, nil
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

	if p.TracksOnly {
		return writeTracklist(results, tracksPath(output))
	}

	if p.DryRun {
		return nil
	}

	if len(p.Just) > 0 {
		return nil
	}

	var recipes []audio.Recipe
	for _, sr := range results {
		if sr.recipe.InputPath != "" {
			recipes = append(recipes, sr.recipe)
		}
	}

	if err := audio.ConcatRecipes(recipes, output); err != nil {
		return err
	}

	return writeTracklist(results, tracksPath(output))
}

func tracksPath(output string) string {
	return output[:len(output)-len(filepath.Ext(output))] + ".txt"
}

func writeTracklist(results []sliceResult, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, sr := range results {
		if sr.recipe.InputPath == "" {
			continue
		}
		fmt.Fprintln(f, trackLabel(sr)) //nolint:errcheck
	}
	fmt.Println("tracks:", path)
	return nil
}

func shouldProcess(just []int, i int) bool {
	return len(just) == 0 || slices.Contains(just, i)
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
