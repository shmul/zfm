package m3u

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Track struct {
	Length string
	Title  string
	Path   string
}

// Parse reads an EXTM3U file and returns the track list.
func Parse(path string) ([]Track, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	scanner.Scan()
	if first := scanner.Text(); !strings.HasPrefix(first, "#EXTM3U") {
		return nil, fmt.Errorf("not an EXTM3U file: %q", path)
	}

	var (
		tracks  []Track
		current Track
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "#EXTINF:"); ok {
			length, title, _ := strings.Cut(rest, ",")
			current = Track{Length: strings.TrimSpace(length), Title: strings.TrimSpace(title)}
		} else if !strings.HasPrefix(line, "#") {
			current.Path = line
			tracks = append(tracks, current)
			current = Track{}
		}
	}
	return tracks, scanner.Err()
}
