package mix

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"
)

func generateSilentMP3(t *testing.T, dir, name string, secs int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", fmt.Sprintf("anullsrc=r=44100:cl=mono"),
		"-t", fmt.Sprintf("%d", secs),
		"-q:a", "9", "-acodec", "libmp3lame",
		"-y", path,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return path
}

func writeMixFileContent(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "test.mix")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestRunner(t *testing.T) {
	bdd.Scenario(t, "Dry-run", func(t *testing.T, _ string) {
		bdd.Test(t, "prints recipe without writing files", func() {
			dir := t.TempDir()
			track := generateSilentMP3(t, dir, "a.mp3", 5)

			mixPath := writeMixFileContent(t, dir, fmt.Sprintf(`
[tracks]
a = %q

[[mix]]
track    = "a"
fade_in  = 0.5
fade_out = 0.5
`, track))

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := Run(Params{MixFile: mixPath, Just: -1, DryRun: true})

			w.Close()
			os.Stdout = old
			var buf bytes.Buffer
			io.Copy(&buf, r) //nolint:errcheck

			require.NoError(t, err)
			require.Contains(t, buf.String(), "[dry-run]")

			// No output MP3 should have been created
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				require.NotEqual(t, "playlist.mp3", e.Name())
			}
		})
	})

	bdd.Scenario(t, "Just flag", func(t *testing.T, _ string) {
		bdd.Test(t, "just=1 processes only the second slice", func() {
			dir := t.TempDir()
			trackA := generateSilentMP3(t, dir, "a.mp3", 3)
			trackB := generateSilentMP3(t, dir, "b.mp3", 3)

			mixPath := writeMixFileContent(t, dir, fmt.Sprintf(`
[tracks]
a = %q
b = %q

[[mix]]
track = "a"

[[mix]]
track   = "b"
fade_in = 0.5
`, trackA, trackB))

			err := Run(Params{MixFile: mixPath, Just: 1, DryRun: false})
			require.NoError(t, err)

			// just=1 skips concat — no playlist written
			_, statErr := os.Stat(filepath.Join(dir, "playlist.mp3"))
			require.True(t, os.IsNotExist(statErr))
		})
	})
}
