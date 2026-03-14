package audio

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

func generateSilentMP3(t *testing.T, secs int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "silent.mp3")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono",
		"-t", fmt.Sprintf("%d", secs),
		"-q:a", "9", "-acodec", "libmp3lame",
		"-y", path,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return path
}

func TestParseTime(t *testing.T) {
	bdd.Scenario(t, "ParseTime converts time strings to seconds", func(t *testing.T, _ string) {
		bdd.Test(t, "empty string returns 0", func() {
			v, err := ParseTime("")
			require.NoError(t, err)
			require.Equal(t, 0.0, v)
		})

		bdd.Test(t, "mm:ss format", func() {
			v, err := ParseTime("1:30")
			require.NoError(t, err)
			require.Equal(t, 90.0, v)
		})

		bdd.Test(t, "hh:mm:ss format", func() {
			v, err := ParseTime("1:02:03")
			require.NoError(t, err)
			require.Equal(t, 3723.0, v)
		})

		bdd.Test(t, "zero values", func() {
			v, err := ParseTime("0:00")
			require.NoError(t, err)
			require.Equal(t, 0.0, v)
		})

		bdd.Test(t, "invalid format returns error", func() {
			_, err := ParseTime("notatime")
			require.Error(t, err)
		})

		bdd.Test(t, "sub-second input", func() {
			v, err := ParseTime("1:30.5")
			require.NoError(t, err)
			require.Equal(t, 90.5, v)
		})
	})
}

func TestBuildAudioFilter(t *testing.T) {
	bdd.Scenario(t, "buildAudioFilter constructs correct filter chains", func(t *testing.T, _ string) {
		bdd.Test(t, "omits volume filter when zero", func() {
			f := buildAudioFilter(10, 1, 1, "qsin", 0)
			require.NotContains(t, f, "volume=")
			require.Contains(t, f, "afade=t=in")
		})

		bdd.Test(t, "appends volume filter when non-zero", func() {
			f := buildAudioFilter(10, 0, 0, "qsin", -6.0)
			require.Contains(t, f, "volume=-6.0000dB")
		})

		bdd.Test(t, "filter order: asetpts → afade in → afade out → volume", func() {
			f := buildAudioFilter(10, 1, 1, "qsin", -3.0)
			asetpts := strings.Index(f, "asetpts")
			fadeIn := strings.Index(f, "afade=t=in")
			fadeOut := strings.Index(f, "afade=t=out")
			vol := strings.Index(f, "volume=")
			require.Less(t, asetpts, fadeIn)
			require.Less(t, fadeIn, fadeOut)
			require.Less(t, fadeOut, vol)
		})
	})
}

func TestIdenticalWithVolume(t *testing.T) {
	bdd.Scenario(t, "Identical flag respects Volume", func(t *testing.T, _ string) {
		bdd.Test(t, "Identical=false when Volume != 0 and no crop or fades", func() {
			r := Recipe{SS: 0, To: 60, FadeIn: 0, FadeOut: 0, Volume: -3.0, Duration: 60}
			identical := r.SS == 0 && r.To == r.Duration && r.FadeIn == 0 && r.FadeOut == 0 && r.Volume == 0
			require.False(t, identical)
		})

		bdd.Test(t, "Identical=true when Volume == 0 and no crop or fades", func() {
			r := Recipe{SS: 0, To: 60, FadeIn: 0, FadeOut: 0, Volume: 0, Duration: 60}
			identical := r.SS == 0 && r.To == r.Duration && r.FadeIn == 0 && r.FadeOut == 0 && r.Volume == 0
			require.True(t, identical)
		})
	})
}

func TestPrepareTargetVolume(t *testing.T) {
	bdd.Scenario(t, "Prepare resolves TargetVolume to Recipe.Volume", func(t *testing.T, _ string) {
		bdd.Test(t, "TargetVolume nil leaves Volume unchanged", func() {
			path := generateSilentMP3(t, 3)
			r, _, err := Prepare(PrepareParams{Path: path})
			require.NoError(t, err)
			require.Equal(t, 0.0, r.Volume)
		})

		bdd.Test(t, "TargetVolume set computes gain as target minus measured mean", func() {
			path := generateSilentMP3(t, 3)
			// Measure the mean ourselves to compute the expected gain.
			vol, err := VolumeStatsRange(path, 0, 3)
			require.NoError(t, err)

			target := -16.0
			r, _, err := Prepare(PrepareParams{Path: path, TargetVolume: &target})
			require.NoError(t, err)
			require.InDelta(t, target-vol.Mean, r.Volume, 0.01)
		})
	})
}

func TestEndOffset(t *testing.T) {
	bdd.Scenario(t, "endOffset resolves end position from various inputs", func(t *testing.T, _ string) {
		const dur = 120.0

		bdd.Test(t, "empty hms and zero tail returns full duration", func() {
			v, err := endOffset("", 0, dur)
			require.NoError(t, err)
			require.Equal(t, dur, v)
		})

		bdd.Test(t, "tail trims from end", func() {
			v, err := endOffset("", 10, dur)
			require.NoError(t, err)
			require.Equal(t, 110.0, v)
		})

		bdd.Test(t, "tail larger than duration clamps to 0", func() {
			v, err := endOffset("", 200, dur)
			require.NoError(t, err)
			require.Equal(t, 0.0, v)
		})

		bdd.Test(t, "explicit end time overrides tail", func() {
			v, err := endOffset("1:00", 10, dur)
			require.NoError(t, err)
			require.Equal(t, 60.0, v)
		})
	})
}
