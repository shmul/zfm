package audio

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/frioux/leatherman/pkg/shellquote"
	"github.com/ionrock/procs"
	"github.com/rs/zerolog/log"
)

// newCmd logs the command string at debug level and returns a ready Process.
func newCmd(cmd string) *procs.Process {
	log.Debug().Str("cmd", cmd).Msg("exec")
	return procs.NewProcess(cmd)
}

type (
	VolumeInfo struct {
		Mean float64 // mean_volume in dBFS
		Peak float64 // max_volume in dBFS
	}

	ProfileWindow struct {
		SS     float64 // window start in source file (seconds)
		Volume VolumeInfo
	}
)

var (
	reMean = regexp.MustCompile(`mean_volume:\s*([-\d.]+)\s*dB`)
	rePeak = regexp.MustCompile(`max_volume:\s*([-\d.]+)\s*dB`)
)

// VolumeStats runs ffmpeg volumedetect on path and returns mean/peak dBFS.
func VolumeStats(path string) (VolumeInfo, error) {
	return volumeDetect(path, -1, -1)
}

// VolumeStatsRange runs volumedetect on a sub-range [ss, to] of path.
func VolumeStatsRange(path string, ss, to float64) (VolumeInfo, error) {
	return volumeDetect(path, ss, to)
}

func volumeDetect(path string, ss, to float64) (VolumeInfo, error) {
	var args []string
	if ss >= 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", ss))
	}
	if to >= 0 {
		args = append(args, "-to", fmt.Sprintf("%.3f", to))
	}
	args = append(args, "-i", path, "-af", "volumedetect", "-f", "null", "-")

	cmd, err := ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return VolumeInfo{}, err
	}
	p := newCmd(cmd)
	runErr := p.Run()
	errBytes, _ := p.ErrOutput()
	if runErr != nil && len(errBytes) == 0 {
		return VolumeInfo{}, runErr
	}

	out := string(errBytes)
	mean, err := parseDBFS(reMean, out)
	if err != nil {
		return VolumeInfo{}, fmt.Errorf("mean_volume: %w", err)
	}
	peak, err := parseDBFS(rePeak, out)
	if err != nil {
		return VolumeInfo{}, fmt.Errorf("max_volume: %w", err)
	}

	return VolumeInfo{Mean: mean, Peak: peak}, nil
}

// ProcsCmdStr builds a shell-quoted command string for procs.NewProcess.
func ProcsCmdStr(name string, args []string) (string, error) {
	return shellquote.Quote(append([]string{name}, args...))
}

func parseDBFS(re *regexp.Regexp, s string) (float64, error) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return math.Inf(-1), nil
	}
	return strconv.ParseFloat(strings.TrimSpace(m[1]), 64)
}

// VolumeProfile returns per-second volume stats for [ss, to] in path.
// Windows run in parallel; each is 1 second wide.
func VolumeProfile(path string, ss, to float64) ([]ProfileWindow, error) {
	const windowSecs = 1.0
	n := int(math.Ceil((to - ss) / windowSecs))
	results := make([]ProfileWindow, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		start := ss + float64(i)*windowSecs
		end := math.Min(start+windowSecs, to)
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := VolumeStatsRange(path, start, end)
			results[i] = ProfileWindow{SS: start, Volume: v}
			errs[i] = err
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// FormatProfile renders a per-second volume profile with threshold markers and
// a suggested --tail value. sourceDuration is used to compute "X from end" labels.
func FormatProfile(profile []ProfileWindow, sourceDuration, threshold float64) string {
	var sb strings.Builder

	lastLoudIdx := -1
	for i, w := range profile {
		if w.Volume.Mean > threshold {
			lastLoudIdx = i
		}
		fromEnd := sourceDuration - w.SS
		marker := ""
		if w.Volume.Mean <= threshold {
			marker = "  [quiet]"
		}
		fmt.Fprintf(&sb, "    -%4.1fs  %5.1f / %5.1f dBFS%s\n",
			fromEnd, w.Volume.Mean, w.Volume.Peak, marker)
	}

	if lastLoudIdx >= 0 {
		const windowSecs = 1.0
		silenceStartSS := profile[lastLoudIdx].SS + windowSecs
		suggestedTail := sourceDuration - silenceStartSS
		fmt.Fprintf(&sb, "  suggested: --tail %.1f\n", suggestedTail)
	}

	return sb.String()
}
