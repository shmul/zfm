package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ionrock/procs"
	"github.com/rs/zerolog/log"
	"github.com/shmul/zfm/internal/audio"
	"golang.org/x/sync/errgroup"
)

type (
	Params struct {
		CSVFile       string
		TargetDir     string
		Preview       float64
		Just          int
		OneByOne      bool
		DryRun        bool
		SilenceThresh float64
	}

	trackResult struct {
		idx      int
		artist   string
		title    string
		duration float64 // seconds after crop
		destPath string
		recipe   audio.Recipe
		skip     bool
	}
)

func Run(p Params) error {
	destDir := p.TargetDir
	if destDir == "" {
		destDir = filepath.Dir(p.CSVFile)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	rows, err := readCSV(p.CSVFile)
	if err != nil {
		return err
	}

	results := make([]trackResult, len(rows))
	var mu sync.Mutex
	var overallDur float64

	workers := runtime.GOMAXPROCS(0)
	g, _ := errgroup.WithContext(context.Background())
	sem := make(chan struct{}, workers)

	for idx, row := range rows {
		if p.Just >= 0 && idx != p.Just {
			continue
		}
		idx, row := idx, row

		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()

			r, info, err := processRow(row, destDir, p.DryRun || p.Preview > 0)
			if err != nil {
				log.Warn().Err(err).Int("idx", idx).Msg("csv - process row")
				return nil // non-fatal: skip bad rows
			}

			tr := buildTrackResult(idx, row, r, info)
			mu.Lock()
			results[idx] = tr
			overallDur += tr.duration
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	printTracklist(results, p.Just)

	fmt.Println("overall time", fmtDuration(overallDur))

	if err := writeTracklist(destDir, results); err != nil {
		return err
	}

	if p.Preview > 0 {
		for _, tr := range results {
			if tr.destPath == "" {
				continue
			}
			previewTrack(tr, p.Preview, p.SilenceThresh)
		}
		return nil
	}

	if !p.DryRun && p.Just < 0 && !p.OneByOne {
		return concatPlaylist(destDir, results)
	}

	if p.OneByOne {
		for _, tr := range results {
			if tr.destPath == "" || tr.skip {
				continue
			}
			if err := audio.PlayFile(tr.destPath); err != nil {
				log.Warn().Err(err).Str("path", tr.destPath).Msg("csv - play")
			}
		}
	}
	return nil
}

func processRow(row map[string]string, destDir string, dryRun bool) (audio.Recipe, audio.ProbeInfo, error) {
	rawPath := row["file"]
	path, err := url.PathUnescape(rawPath)
	if err != nil {
		path = rawPath
	}
	if strings.HasPrefix(path, "file://") {
		if u, err := url.Parse(path); err == nil {
			path = u.Path
		}
	}

	r, info, err := audio.Prepare(audio.PrepareParams{
		Path: path, Start: row["start"], End: row["end"],
		Head: parseFloat(row["head"]), Tail: parseFloat(row["tail"]),
		FadeIn: parseFloat(row["fade_in"]), FadeOut: parseFloat(row["fade_out"]),
		FadeCurve: row["fade_curve"],
	})
	if err != nil {
		return audio.Recipe{}, audio.ProbeInfo{}, err
	}

	if !dryRun {
		dest := filepath.Join(destDir, filepath.Base(path))
		if err := audio.Execute(r, dest); err != nil {
			return r, info, err
		}
		r.InputPath = dest
	}

	return r, info, nil
}

func buildTrackResult(idx int, row map[string]string, r audio.Recipe, info audio.ProbeInfo) trackResult {
	artist := info.Tags["ARTIST"]
	title := info.Tags["TITLE"]
	skip := artist == "" && title == ""
	if artist == "" {
		artist = filepath.Base(row["file"])
	}

	dur := r.To - r.SS
	if dur <= 0 {
		dur = r.Duration
	}

	return trackResult{
		idx:      idx,
		artist:   artist,
		title:    title,
		duration: dur,
		destPath: r.InputPath,
		recipe:   r,
		skip:     skip,
	}
}

func printTracklist(results []trackResult, just int) {
	var acc float64
	for _, tr := range results {
		if tr.destPath == "" {
			continue
		}
		line := fmt.Sprintf("(%s) [%s] %s", fmtDuration(acc), fmtDuration(tr.duration), tr.artist)
		if tr.title != "" {
			line += " - " + tr.title
		}
		fmt.Println(line)
		acc += tr.duration
	}
}

func writeTracklist(destDir string, results []trackResult) error {
	f, err := os.Create(filepath.Join(destDir, "tracklist.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	for _, tr := range results {
		if tr.skip || tr.destPath == "" {
			continue
		}
		line := tr.artist
		if tr.title != "" {
			line += " - " + tr.title
		}
		fmt.Fprintln(f, line)
	}
	return nil
}

func concatPlaylist(destDir string, results []trackResult) error {
	listFile := filepath.Join(destDir, "concat.txt")
	f, err := os.Create(listFile)
	if err != nil {
		return err
	}
	for _, tr := range results {
		if tr.destPath == "" {
			continue
		}
		abs, _ := filepath.Abs(tr.destPath)
		fmt.Fprintf(f, "file '%s'\n", strings.ReplaceAll(abs, "'", "'\\''"))
	}
	f.Close()

	target := filepath.Join(destDir, "playlist.mp3")
	fmt.Println(target)

	args := []string{"-f", "concat", "-safe", "0", "-i", listFile, "-b:a", "320k", "-y", target}
	cmd, err := audio.ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return err
	}
	p := procs.NewProcess(cmd)
	p.ErrHandler = func(line string) string { fmt.Fprintln(os.Stderr, line); return line }
	return p.Run()
}

func previewTrack(tr trackResult, previewSecs, silenceThresh float64) {
	fmt.Printf("\n==== %02d [%s] %s", tr.idx, fmtDuration(tr.duration), tr.artist)
	if tr.title != "" {
		fmt.Printf(" - %s", tr.title)
	}
	fmt.Println()

	r := tr.recipe

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

	if profile, err := audio.VolumeProfile(r.InputPath, tailR.SS, r.To); err == nil {
		fmt.Print(audio.FormatProfile(profile, r.To, silenceThresh))
	}

	audio.Play(tailR, "tail:") //nolint:errcheck
}

func isKVCell(v string) bool {
	if !strings.Contains(v, "=") || strings.HasPrefix(v, "=") {
		return false
	}
	parts := strings.SplitN(v, "=", 2)
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func parseRecord(record, header []string) map[string]string {
	row := make(map[string]string, len(header))
	kvMode := false
	for _, v := range record {
		if isKVCell(v) {
			kvMode = true
			break
		}
	}
	if kvMode {
		for _, v := range record {
			v = strings.TrimSpace(v)
			if isKVCell(v) {
				parts := strings.SplitN(v, "=", 2)
				row[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		// file path is positional (cell 0) when it doesn't look like a kv pair
		if len(record) > 0 {
			if first := strings.TrimSpace(record[0]); !strings.Contains(first, "=") {
				row["file"] = first
			}
		}
	} else {
		for i, h := range header {
			if i < len(record) {
				row[h] = strings.TrimSpace(record[i])
			}
		}
	}
	return row
}

func readCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // variable fields
	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	var rows []map[string]string
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		rows = append(rows, parseRecord(record, header))
	}
	return rows, nil
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func fmtDuration(secs float64) string {
	d := time.Duration(secs) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
