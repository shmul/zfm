package mix

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type (
	Slice struct {
		Track     string  `toml:"track"`
		SS        string  `toml:"ss"`
		To        string  `toml:"to"`
		Head      float64 `toml:"head"`
		Tail      float64 `toml:"tail"`
		FadeIn    float64 `toml:"fade_in"`
		FadeOut   float64 `toml:"fade_out"`
		FadeCurve string  `toml:"fade_curve"`
	}

	MixFile struct {
		Output string            `toml:"output"`
		Tracks map[string]string `toml:"tracks"`
		Mix    []Slice           `toml:"mix"`
	}
)

// Parse decodes a .mix TOML file, enforces strict key validation and reference integrity.
func Parse(path string) (MixFile, error) {
	var mf MixFile
	meta, err := toml.DecodeFile(path, &mf)
	if err != nil {
		return MixFile{}, err
	}

	if unknown := meta.Undecoded(); len(unknown) > 0 {
		keys := make([]string, len(unknown))
		for i, k := range unknown {
			keys[i] = k.String()
		}
		return MixFile{}, fmt.Errorf("unknown keys in mix file: %s", strings.Join(keys, ", "))
	}

	if mf.Output == "" {
		mf.Output = "playlist.mp3"
	}

	base := filepath.Dir(path)
	for name, p := range mf.Tracks {
		if !filepath.IsAbs(p) {
			mf.Tracks[name] = filepath.Join(base, p)
		}
	}

	var missing []string
	for i, s := range mf.Mix {
		if s.Track == "" {
			return MixFile{}, fmt.Errorf("slice %d: track field is required", i)
		}
		if _, ok := mf.Tracks[s.Track]; !ok {
			missing = append(missing, s.Track)
		}
	}
	if len(missing) > 0 {
		return MixFile{}, fmt.Errorf("undefined track references: %s", strings.Join(missing, ", "))
	}

	return mf, nil
}
