package main

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/shmul/zfm/internal/audio"
	zcsv "github.com/shmul/zfm/internal/csv"
	"github.com/shmul/zfm/internal/m3u"
)

var csvFieldNames = []string{"file", "start", "end", "head", "tail", "fade_in", "fade_out", "fade_curve"}

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
	csvPath := filepath.Join(c.Args.Dir, "playlist.csv")
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

	return filepath.WalkDir(c.Args.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".flac") {
			return w.Write([]string{path, ""})
		}
		return nil
	})
}

func (c *analyzeCmd) Execute(_ []string) error {
	verbose()
	return printSilenceAnalysis(c.Args.Filename, audio.SilenceParams{
		Threshold:        c.SilenceThresh,
		MinSilence:       c.MinSilence,
		AnalysisDuration: c.AnalysisDuration,
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
