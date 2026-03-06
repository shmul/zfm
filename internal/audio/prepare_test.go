package audio

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

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
