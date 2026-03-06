package csv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

func TestReadCSV(t *testing.T) {
	bdd.Scenario(t, "readCSV parses standard and kv-mode rows", func(t *testing.T, _ string) {
		bdd.Test(t, "standard header+row", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.csv")
			require.NoError(t, os.WriteFile(path, []byte(
				"file,start,end,head,tail,fade_in,fade_out\n"+
					"/some/track.mp3,0:30,,0,5,2,3\n",
			), 0o644))

			rows, err := readCSV(path)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, "/some/track.mp3", rows[0]["file"])
			require.Equal(t, "0:30", rows[0]["start"])
			require.Equal(t, "5", rows[0]["tail"])
		})

		bdd.Test(t, "kv-mode row with explicit file key", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "kv.csv")
			require.NoError(t, os.WriteFile(path, []byte(
				"file,start,end,head,tail,fade_in,fade_out\n"+
					"file=/some/track.mp3,tail=10,fade_in=2\n",
			), 0o644))

			rows, err := readCSV(path)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, "/some/track.mp3", rows[0]["file"])
			require.Equal(t, "10", rows[0]["tail"])
			require.Equal(t, "2", rows[0]["fade_in"])
		})

		bdd.Test(t, "kv-mode row with positional file path", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "kv_pos.csv")
			require.NoError(t, os.WriteFile(path, []byte(
				"file,start,end,head,tail,fade_in,fade_out\n"+
					"/some/My Track.mp3,tail=4\n",
			), 0o644))

			rows, err := readCSV(path)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, "/some/My Track.mp3", rows[0]["file"])
			require.Equal(t, "4", rows[0]["tail"])
		})

		bdd.Test(t, "empty file returns no rows", func() {
			dir := t.TempDir()
			path := filepath.Join(dir, "empty.csv")
			require.NoError(t, os.WriteFile(path, []byte(
				"file,start,end,head,tail,fade_in,fade_out\n",
			), 0o644))

			rows, err := readCSV(path)
			require.NoError(t, err)
			require.Empty(t, rows)
		})
	})
}

func TestFmtDuration(t *testing.T) {
	bdd.Scenario(t, "fmtDuration formats seconds to mm:ss or h:mm:ss", func(t *testing.T, _ string) {
		bdd.Test(t, "under one hour", func() {
			require.Equal(t, "3:05", fmtDuration(185))
		})
		bdd.Test(t, "exactly one hour", func() {
			require.Equal(t, "1:00:00", fmtDuration(3600))
		})
		bdd.Test(t, "zero", func() {
			require.Equal(t, "0:00", fmtDuration(0))
		})
	})
}
