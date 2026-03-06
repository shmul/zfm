package audio

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type (
	SilenceParams struct {
		Threshold        float64 // dBFS, e.g. -40.0
		MinSilence       float64 // seconds
		AnalysisDuration float64 // seconds to analyze from end (0 = full file)
	}

	SilenceSegment struct {
		Start float64 // seconds from start of file
		End   float64
	}

	SilenceAnalysis struct {
		TotalDuration float64
		Mean          float64
		Peak          float64
		Segments      []SilenceSegment
		Profile       []ProfileWindow
		Threshold     float64
		// SuggestedTrimFromEnd is > 0 when a silence segment reaches the file end.
		SuggestedTrimFromEnd float64
	}
)

var (
	reSilStart = regexp.MustCompile(`silence_start:\s*([-\d.]+)`)
	reSilEnd   = regexp.MustCompile(`silence_end:\s*([-\d.]+)`)
)

// DetectSilence runs ffmpeg silencedetect on the file and returns analysis results.
func DetectSilence(path string, p SilenceParams) (SilenceAnalysis, error) {
	info, err := Probe(path)
	if err != nil {
		return SilenceAnalysis{}, err
	}

	vol, err := VolumeStats(path)
	if err != nil {
		return SilenceAnalysis{}, err
	}

	filter := fmt.Sprintf("silencedetect=noise=%.1fdB:d=%.3f", p.Threshold, p.MinSilence)
	args := []string{"-i", path}
	if p.AnalysisDuration > 0 {
		ss := math.Max(0, info.Duration-p.AnalysisDuration)
		args = append([]string{"-ss", fmt.Sprintf("%.3f", ss)}, args...)
	}
	args = append(args, "-af", filter, "-f", "null", "-")
	cmd, err := ProcsCmdStr("ffmpeg", args)
	if err != nil {
		return SilenceAnalysis{}, err
	}
	proc := newCmd(cmd)
	proc.Run() //nolint:errcheck // non-zero exit is expected for -f null
	errBytes, _ := proc.ErrOutput()

	tailOffset := 0.0
	if p.AnalysisDuration > 0 {
		tailOffset = math.Max(0, info.Duration-p.AnalysisDuration)
	}
	segments := parseSilenceSegments(string(errBytes), tailOffset)

	var trimFromEnd float64
	const endTolerance = 0.1
	for _, s := range segments {
		if s.End >= info.Duration-endTolerance {
			trimFromEnd = info.Duration - s.Start
			break
		}
	}

	profile, _ := VolumeProfile(path, tailOffset, info.Duration)

	return SilenceAnalysis{
		TotalDuration:        info.Duration,
		Mean:                 vol.Mean,
		Peak:                 vol.Peak,
		Segments:             segments,
		Profile:              profile,
		Threshold:            p.Threshold,
		SuggestedTrimFromEnd: trimFromEnd,
	}, nil
}

func parseSilenceSegments(output string, offset float64) []SilenceSegment {
	starts := parseFloats(reSilStart, output)
	ends := parseFloats(reSilEnd, output)

	segments := make([]SilenceSegment, 0, len(starts))
	for i, s := range starts {
		end := math.Inf(1)
		if i < len(ends) {
			end = ends[i] + offset
		}
		segments = append(segments, SilenceSegment{Start: s + offset, End: end})
	}
	return segments
}

func parseFloats(re *regexp.Regexp, s string) []float64 {
	matches := re.FindAllStringSubmatch(s, -1)
	out := make([]float64, 0, len(matches))
	for _, m := range matches {
		if v, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// FormatSilenceAnalysis returns a human-readable summary.
func FormatSilenceAnalysis(a SilenceAnalysis) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Audio duration: %.1fs\n", a.TotalDuration)
	fmt.Fprintf(&sb, "Tail volume: %.1f / %.1f dBFS\n", a.Mean, a.Peak)

	if len(a.Segments) == 0 {
		sb.WriteString("No silence segments found in tail\n")
	} else {
		fmt.Fprintf(&sb, "Found %d silence segment(s):\n", len(a.Segments))
		for i, s := range a.Segments {
			dur := s.End - s.Start
			fmt.Fprintf(&sb, "  %d. %.1fs - %.1fs (%.1fs)\n", i+1, s.Start, s.End, dur)
		}
	}

	if len(a.Profile) > 0 {
		sb.WriteString("Volume profile:\n")
		sb.WriteString(FormatProfile(a.Profile, a.TotalDuration, a.Threshold))
	} else if a.SuggestedTrimFromEnd > 0 {
		fmt.Fprintf(&sb, "Suggested trim: %.1fs from end\n", a.SuggestedTrimFromEnd)
	}
	return sb.String()
}
