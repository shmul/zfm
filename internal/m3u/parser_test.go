package m3u

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

func TestParse(t *testing.T) {
	bdd.Scenario(t, "Parse reads EXTM3U playlists", func(t *testing.T, _ string) {
		bdd.Test(t, "standard playlist with two tracks", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.m3u")
			require.NoError(t, os.WriteFile(path, []byte(
				"#EXTM3U\n"+
					"#EXTINF:419,Alice In Chains - Rotten Apple\n"+
					"/music/rotten_apple.mp3\n"+
					"#EXTINF:253,Soundgarden - Black Hole Sun\n"+
					"/music/black_hole_sun.mp3\n",
			), 0o644))

			tracks, err := Parse(path)
			require.NoError(t, err)
			require.Len(t, tracks, 2)
			require.Equal(t, "419", tracks[0].Length)
			require.Equal(t, "Alice In Chains - Rotten Apple", tracks[0].Title)
			require.Equal(t, "/music/rotten_apple.mp3", tracks[0].Path)
			require.Equal(t, "/music/black_hole_sun.mp3", tracks[1].Path)
		})

		bdd.Test(t, "missing EXTM3U header returns error", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "bad.m3u")
			require.NoError(t, os.WriteFile(path, []byte("/music/track.mp3\n"), 0o644))

			_, err := Parse(path)
			require.Error(t, err)
		})

		bdd.Test(t, "plain path without EXTINF is included", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "simple.m3u")
			require.NoError(t, os.WriteFile(path, []byte(
				"#EXTM3U\n/music/track.mp3\n",
			), 0o644))

			tracks, err := Parse(path)
			require.NoError(t, err)
			require.Len(t, tracks, 1)
			require.Equal(t, "/music/track.mp3", tracks[0].Path)
		})

		bdd.Test(t, "empty playlist returns no tracks", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "empty.m3u")
			require.NoError(t, os.WriteFile(path, []byte("#EXTM3U\n"), 0o644))

			tracks, err := Parse(path)
			require.NoError(t, err)
			require.Empty(t, tracks)
		})
	})
}
