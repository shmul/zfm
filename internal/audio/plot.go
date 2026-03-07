package audio

import (
	"fmt"
	"math"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/charmbracelet/x/term"
)

const dbFloor = -70.0

var (
	loudBlock  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("2"))
	quietBlock = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Background(lipgloss.Color("1"))
	peakBlock  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Background(lipgloss.Color("3"))
)

// PlotProfile renders a per-second volume profile as a terminal bar chart.
// Each bar shows mean volume (green/red depending on threshold) with peak delta on top (yellow).
func PlotProfile(profile []ProfileWindow, threshold float64) string {
	if len(profile) == 0 {
		return ""
	}

	const chartHeight = 12
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || w < 20 {
		w = 80
	}

	bc := barchart.New(w, chartHeight,
		barchart.WithMaxValue(-dbFloor),
		barchart.WithNoAutoMaxValue(),
		barchart.WithBarGap(0),
	)

	for _, win := range profile {
		meanH := math.Max(0, win.Volume.Mean-dbFloor)
		peakH := math.Max(0, win.Volume.Peak-win.Volume.Mean)
		meanSt := loudBlock
		if win.Volume.Mean <= threshold {
			meanSt = quietBlock
		}
		bc.Push(barchart.BarData{
			Values: []barchart.BarValue{
				{Value: meanH, Style: meanSt},
				{Value: peakH, Style: peakBlock},
			},
		})
	}

	bc.Draw()

	threshLabel := fmt.Sprintf("%.0fdB", threshold)
	header := fmt.Sprintf("  0dBFS  %s mean  %s peak  %s quiet (≤%s)\n",
		loudBlock.Render("■"),
		peakBlock.Render("■"),
		quietBlock.Render("■"),
		threshLabel,
	)
	footer := fmt.Sprintf("%.0fdBFS  %s  (%ds)\n",
		dbFloor,
		strings.Repeat("─", w/3),
		len(profile),
	)

	return header + bc.View() + "\n" + footer
}
