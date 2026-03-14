package mix

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
	"github.com/themakers/bdd"

	"github.com/shmul/zfm/internal/audio"
)

func newPlotModel(volAdj float64) plotModel {
	return plotModel{
		entries:   []plotEntry{{label: "test", recipe: audio.Recipe{SS: 0, To: 10, Duration: 10}, volAdj: volAdj}},
		positions: []float64{0},
		threshold: -50,
	}
}

func pressKey(m plotModel, key string) plotModel {
	updated, _ := m.Update(tea.KeyPressMsg{Text: key})
	return updated.(plotModel)
}

func TestPlotVolAdj(t *testing.T) {
	bdd.Scenario(t, "Plot TUI volume adjustment keybindings", func(t *testing.T, _ string) {
		bdd.Test(t, "+ increments volAdj by 1.0", func() {
			m := pressKey(newPlotModel(0), "+")
			require.Equal(t, 1.0, m.entries[0].volAdj)
		})

		bdd.Test(t, "= also increments volAdj by 1.0", func() {
			m := pressKey(newPlotModel(0), "=")
			require.Equal(t, 1.0, m.entries[0].volAdj)
		})

		bdd.Test(t, "- decrements volAdj by 1.0", func() {
			m := pressKey(newPlotModel(0), "-")
			require.Equal(t, -1.0, m.entries[0].volAdj)
		})

		bdd.Test(t, "header shows vol offset when non-zero", func() {
			view := newPlotModel(-3.0).View().Content
			require.Contains(t, view, "vol: -3.0 dB")
		})

		bdd.Test(t, "header omits vol when zero", func() {
			view := newPlotModel(0).View().Content
			require.NotContains(t, view, "vol:")
		})

		bdd.Test(t, "nav hint always shows +/-:vol", func() {
			view := newPlotModel(0).View().Content
			require.True(t, strings.Contains(view, "+/-:vol"))
		})
	})
}
