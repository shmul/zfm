package audio

import (
	"encoding/json"
	"strconv"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type (
	ProbeInfo struct {
		Duration float64
		Tags     map[string]string
	}

	probeResult struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Tags map[string]string `json:"tags"`
		} `json:"streams"`
	}
)

func Probe(path string) (ProbeInfo, error) {
	out, err := ffmpeg.Probe(path)
	if err != nil {
		return ProbeInfo{}, err
	}

	var result probeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return ProbeInfo{}, err
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return ProbeInfo{}, err
	}

	tags := result.Format.Tags
	if len(tags) == 0 && len(result.Streams) > 0 {
		tags = result.Streams[0].Tags
	}

	// normalise tag keys to uppercase
	normalised := make(map[string]string, len(tags))
	for k, v := range tags {
		upper := ""
		for _, r := range k {
			if r >= 'a' && r <= 'z' {
				upper += string(r - 32)
			} else {
				upper += string(r)
			}
		}
		normalised[upper] = v
	}

	return ProbeInfo{Duration: duration, Tags: normalised}, nil
}
