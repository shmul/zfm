package mix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

func writeMixFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mix")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestParser(t *testing.T) {
	bdd.Scenario(t, "Parse valid mix file", func(t *testing.T, _ string) {
		bdd.Test(t, "parses all fields correctly", func() {
			path := writeMixFile(t, `
output = "my_set.mp3"

[tracks]
intro = "/music/intro.flac"
main  = "/music/main.mp3"

[[mix]]
track     = "intro"
to        = "0:45"
fade_out  = 2.0

[[mix]]
track      = "main"
ss         = "1:00"
to         = "5:30"
fade_in    = 1.5
fade_out   = 2.0
fade_curve = "tri"
`)
			mf, err := Parse(path)
			require.NoError(t, err)
			require.Equal(t, "my_set.mp3", mf.Output)
			require.Equal(t, "/music/intro.flac", mf.Tracks["intro"])
			require.Equal(t, "/music/main.mp3", mf.Tracks["main"])
			require.Len(t, mf.Mix, 2)
			require.Equal(t, "intro", mf.Mix[0].Track)
			require.Equal(t, "0:45", mf.Mix[0].To)
			require.Equal(t, 2.0, mf.Mix[0].FadeOut)
			require.Equal(t, "main", mf.Mix[1].Track)
			require.Equal(t, "1:00", mf.Mix[1].SS)
			require.Equal(t, 1.5, mf.Mix[1].FadeIn)
			require.Equal(t, "tri", mf.Mix[1].FadeCurve)
		})
	})

	bdd.Scenario(t, "Output defaults", func(t *testing.T, _ string) {
		bdd.Test(t, "defaults output to playlist.mp3 when absent", func() {
			path := writeMixFile(t, `
[tracks]
a = "/music/a.mp3"

[[mix]]
track = "a"
`)
			mf, err := Parse(path)
			require.NoError(t, err)
			require.Equal(t, "playlist.mp3", mf.Output)
		})
	})

	bdd.Scenario(t, "Relative path resolution", func(t *testing.T, _ string) {
		bdd.Test(t, "resolves relative track paths against mix file dir", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "set.mix")
			require.NoError(t, os.WriteFile(path, []byte(`
[tracks]
a = "music/a.mp3"

[[mix]]
track = "a"
`), 0o644))
			mf, err := Parse(path)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, "music/a.mp3"), mf.Tracks["a"])
		})
	})

	bdd.Scenario(t, "Unknown keys", func(t *testing.T, _ string) {
		bdd.Test(t, "errors on unknown top-level key", func() {
			path := writeMixFile(t, `
bogus = "value"

[tracks]
a = "/music/a.mp3"

[[mix]]
track = "a"
`)
			_, err := Parse(path)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unknown keys")
		})

		bdd.Test(t, "errors on unknown key inside [[mix]] slice", func() {
			path := writeMixFile(t, `
[tracks]
a = "/music/a.mp3"

[[mix]]
track   = "a"
mystery = "value"
`)
			_, err := Parse(path)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unknown keys")
		})
	})

	bdd.Scenario(t, "Validation errors", func(t *testing.T, _ string) {
		bdd.Test(t, "errors on missing track field in slice", func() {
			path := writeMixFile(t, `
[tracks]
a = "/music/a.mp3"

[[mix]]
ss = "0:10"
`)
			_, err := Parse(path)
			require.Error(t, err)
			require.Contains(t, err.Error(), "track field is required")
		})

		bdd.Test(t, "errors on reference to undefined track name", func() {
			path := writeMixFile(t, `
[tracks]
a = "/music/a.mp3"

[[mix]]
track = "b"
`)
			_, err := Parse(path)
			require.Error(t, err)
			require.Contains(t, err.Error(), "undefined track references")
			require.Contains(t, err.Error(), "b")
		})
	})
}
