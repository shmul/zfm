package main

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/shmul/zfm/internal/audio"
	zcsv "github.com/shmul/zfm/internal/csv"
	"github.com/shmul/zfm/internal/m3u"
	"github.com/shmul/zfm/internal/mix"
)

var (
	csvFieldNames = []string{"file", "start", "end", "head", "tail", "fade_in", "fade_out", "fade_curve"}
	trackKeyNorm  = strings.NewReplacer(
		"\u2018", "", "\u2019", "", // curly single quotes → drop
		"\u201c", "", "\u201d", "", // curly double quotes → drop
		"\u2013", "-", "\u2014", "-", // en/em dash → hyphen
	)
	supportedAudioExts = []string{".mp3", ".m4a", ".flac", ".aifc", ".wav", ".ogg", ".opus"}
)

func (c *cropCmd) Execute(_ []string) error {
	verbose()
	fmt.Println("zfm crop")
	r, _, err := audio.Prepare(audio.PrepareParams{
		Path: c.Args.Filename, Start: c.Start, End: c.End,
		Head: c.Head, Tail: c.Tail,
		FadeIn: c.FadeIn, FadeOut: c.FadeOut, FadeCurve: c.FadeCurve,
	})
	if err != nil {
		return err
	}

	if c.AnalyzeSilence {
		if err := printSilenceAnalysis(c.Args.Filename, audio.SilenceParams{
			Threshold:  c.SilenceThresh,
			MinSilence: 1.0,
		}); err != nil {
			log.Warn().Err(err).Msg("crop - silence analysis")
		}
	}

	if c.Play {
		return audio.Play(r)
	}

	if c.DryRun {
		fmt.Printf("  [dry-run] %s  ss=%.3f to=%.3f fade_in=%.3f fade_out=%.3f identical=%v\n",
			c.Args.Filename, r.SS, r.To, r.FadeIn, r.FadeOut, r.Identical)
		return nil
	}

	destDir := c.TargetDir
	if destDir == "" {
		destDir = filepath.Dir(c.Args.Filename)
	}
	dest := filepath.Join(destDir, filepath.Base(c.Args.Filename))
	return audio.Execute(r, dest)
}

func (c *spliceCmd) Execute(_ []string) error {
	verbose()
	fmt.Println("zfm splice")

	a, _, err := audio.Prepare(audio.PrepareParams{Path: c.Args.TrackA, End: c.Position})
	if err != nil {
		return err
	}

	b, _, err := audio.Prepare(audio.PrepareParams{Path: c.Args.TrackB})
	if err != nil {
		return err
	}

	return audio.ConcatRecipes([]audio.Recipe{a, b}, c.Output)
}

func (c *playlistCmd) Execute(_ []string) error {
	verbose()
	return zcsv.Run(zcsv.Params{
		CSVFile:       c.Args.Filename,
		TargetDir:     c.TargetDir,
		Preview:       c.Preview,
		Just:          c.Just,
		OneByOne:      c.OneByOne,
		DryRun:        c.DryRun,
		SilenceThresh: c.SilenceThresh,
	})
}

func (c *m3uCmd) Execute(_ []string) error {
	verbose()
	tracks, err := m3u.Parse(c.Args.Filename)
	if err != nil {
		return err
	}

	destDir := c.TargetDir
	if destDir == "" {
		destDir = filepath.Dir(c.Args.Filename)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	base := strings.TrimSuffix(filepath.Base(c.Args.Filename), filepath.Ext(c.Args.Filename))
	csvPath := filepath.Join(destDir, base+".csv")
	fmt.Println("creating", csvPath)

	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(csvFieldNames); err != nil {
		return err
	}
	for _, t := range tracks {
		path, _ := url.PathUnescape(t.Path)
		row := make([]string, len(csvFieldNames))
		row[0] = path
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (c *playCmd) Execute(_ []string) error {
	verbose()
	for _, f := range c.Args.Files {
		r, _, err := audio.Prepare(audio.PrepareParams{
			Path: f, Head: c.Head, Tail: c.Tail,
			FadeIn: c.FadeIn, FadeOut: c.FadeOut, FadeCurve: c.FadeCurve,
		})
		if err != nil {
			return err
		}
		if err := audio.Play(r); err != nil {
			return err
		}
	}
	return nil
}

func (c *generateCmd) Execute(_ []string) error {
	verbose()

	dirBase := filepath.Base(c.Args.Dir)
	var files []string
	if err := filepath.WalkDir(c.Args.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if !lo.Contains(supportedAudioExts, filepath.Ext(lower)) {
			return nil
		}
		// skip generated output files
		if name == "playlist.mp3" || name == dirBase+".mp3" {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return err
	}

	if err := generateCSV(c.Args.Dir, files); err != nil {
		return err
	}
	return generateMixTOML(c.Args.Dir, files)
}

func generateCSV(dir string, files []string) error {
	csvPath := filepath.Join(dir, "playlist.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(csvFieldNames); err != nil {
		return err
	}
	for _, path := range files {
		if err := w.Write([]string{path, ""}); err != nil {
			return err
		}
	}
	return nil
}

func generateMixTOML(dir string, files []string) error {
	base := filepath.Base(dir)
	mixPath := filepath.Join(dir, base+".mix.toml")
	f, err := os.Create(mixPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := &errWriter{w: f}
	w.printf("output = %q\n\n[tracks]\n", base+".mp3")

	keys := make([]string, len(files))
	seen := map[string]int{}
	for i, path := range files {
		name := filepath.Base(path)
		candidate := toTrackKey(strings.TrimSuffix(name, filepath.Ext(name)))
		n := seen[candidate]
		seen[candidate]++
		key := candidate
		if n > 0 {
			key = fmt.Sprintf("%s_%d", candidate, n+1)
		}
		keys[i] = key
		w.printf("%-24s = %q\n", key, name)
	}

	w.printf("\n")
	for _, key := range keys {
		w.printf("[[mix]]\ntrack = %q\ntail = 1\n\n", key)
	}
	return w.err
}

// errWriter accumulates the first write error, allowing caller code to check once at the end.
type errWriter struct {
	w   *os.File
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func toTrackKey(s string) string {
	s = trackKeyNorm.Replace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	key := strings.Trim(b.String(), "_")
	return strings.ReplaceAll(key, "__", "_")
}

func (c *analyzeCmd) Execute(_ []string) error {
	verbose()
	return printSilenceAnalysis(c.Args.Filename, audio.SilenceParams{
		Threshold:        c.SilenceThresh,
		MinSilence:       c.MinSilence,
		AnalysisDuration: c.AnalysisDuration,
	})
}

func (c *mixCmd) Execute(_ []string) error {
	verbose()
	return mix.Run(mix.Params{
		MixFile:       c.Args.Filename,
		TargetDir:     c.TargetDir,
		Preview:       c.Preview,
		Just:          c.Just,
		DryRun:        c.DryRun,
		Plot:          c.Plot,
		TracksOnly:    c.Tracks,
		SilenceThresh: c.SilenceThresh,
	})
}

func printSilenceAnalysis(path string, p audio.SilenceParams) error {
	result, err := audio.DetectSilence(path, p)
	if err != nil {
		return err
	}
	fmt.Print(audio.FormatSilenceAnalysis(result))
	return nil
}
