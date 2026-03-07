package mix

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/shmul/zfm/internal/audio"
)

type (
	plotEntry struct {
		label   string
		recipe  audio.Recipe
		profile []audio.ProfileWindow // nil until loaded
	}

	plotModel struct {
		entries       []plotEntry
		positions     []float64
		idx           int
		threshold     float64
		loading       bool
		playing       bool
		playStartTime time.Time
		playStartPos  float64
		stopPlay      func()
		playDone      <-chan struct{}
		termW         int
		termH         int
	}

	tickMsg struct{}

	profileLoadedMsg struct {
		idx     int
		profile []audio.ProfileWindow
	}

	playDoneMsg struct{}
)

func runPlot(p Params, mf MixFile) error {
	termW, termH, err := term.GetSize(os.Stdout.Fd())
	if err != nil || termW < 20 {
		termW = 80
	}
	if termH < 10 {
		termH = 24
	}

	var entries []plotEntry
	for i, s := range mf.Mix {
		if len(p.Just) > 0 && !slices.Contains(p.Just, i) {
			continue
		}
		r, _, err := audio.Prepare(audio.PrepareParams{
			Path:      mf.Tracks[s.Track],
			Start:     s.Start,
			End:       s.End,
			Head:      s.Head,
			Tail:      s.Tail,
			FadeIn:    s.FadeIn,
			FadeOut:   s.FadeOut,
			FadeCurve: s.FadeCurve,
		})
		if err != nil {
			continue
		}
		dur := r.To - r.SS
		label := fmt.Sprintf("slice %d: %s (%s)", i, s.Track, audio.FmtDuration(dur))
		entries = append(entries, plotEntry{label: label, recipe: r})
	}

	positions := make([]float64, len(entries))
	for i, e := range entries {
		positions[i] = e.recipe.SS
	}

	m := plotModel{entries: entries, positions: positions, threshold: p.SilenceThresh, termW: termW, termH: termH}
	_, err = tea.NewProgram(m).Run()
	return err
}

func (m plotModel) Init() tea.Cmd {
	return m.loadProfile(0)
}

func (m plotModel) loadProfile(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.entries) || m.entries[idx].profile != nil {
		return nil
	}
	r := m.entries[idx].recipe
	termW := m.termW
	return func() tea.Msg {
		dur := r.To - r.SS
		windowSecs := math.Max(1.0, dur/float64(termW))
		profile, err := audio.VolumeProfile(r.InputPath, r.SS, r.To, windowSecs)
		if err != nil {
			return profileLoadedMsg{idx: idx}
		}
		return profileLoadedMsg{idx: idx, profile: profile}
	}
}

func (m plotModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case profileLoadedMsg:
		if msg.idx < len(m.entries) {
			m.entries[msg.idx].profile = msg.profile
		}
		m.loading = false

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.haltAndWait()
			return m, tea.Quit

		case "space":
			if m.playing {
				m.halt()
				return m, nil
			}
			return m, m.play()

		case "[":
			return m, m.seek(-10)
		case "]":
			return m, m.seek(+10)
		case "a":
			return m, m.seekTo(m.curRecipe().SS)
		case "e":
			return m, m.seekTo(m.curRecipe().To)

		case "right", "l", "enter":
			m.halt()
			if m.idx < len(m.entries)-1 {
				m.idx++
				return m, m.triggerLoad()
			}
		case "left", "h":
			m.halt()
			if m.idx > 0 {
				m.idx--
				return m, m.triggerLoad()
			}
		}

	case tickMsg:
		if m.playing {
			elapsed := time.Since(m.playStartTime).Seconds()
			r := m.curRecipe()
			m.positions[m.idx] = math.Min(m.playStartPos+elapsed, r.To)
			return m, doTick()
		}

	case playDoneMsg:
		elapsed := time.Since(m.playStartTime).Seconds()
		r := m.curRecipe()
		m.positions[m.idx] = math.Min(m.playStartPos+elapsed, r.To)
		m.playing = false
		m.stopPlay = nil
	}
	return m, nil
}

func (m *plotModel) triggerLoad() tea.Cmd {
	if m.entries[m.idx].profile != nil {
		return nil
	}
	m.loading = true
	return m.loadProfile(m.idx)
}

func (m *plotModel) halt() {
	if m.stopPlay != nil {
		m.stopPlay()
	}
	m.playing = false
	m.stopPlay = nil
	m.playDone = nil
}

func (m *plotModel) haltAndWait() {
	done := m.playDone
	m.halt()
	if done != nil {
		<-done
	}
}

func (m *plotModel) curRecipe() audio.Recipe {
	if len(m.entries) == 0 {
		return audio.Recipe{}
	}
	return m.entries[m.idx].recipe
}

func (m *plotModel) play() tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	r := m.curRecipe()
	stop, done := audio.StartPlayAt(r.InputPath, m.positions[m.idx], r.To)
	m.playing = true
	m.playStartTime = time.Now()
	m.playStartPos = m.positions[m.idx]
	m.stopPlay = stop
	m.playDone = done
	return tea.Batch(waitForPlay(done), doTick())
}

func doTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *plotModel) seek(delta float64) tea.Cmd {
	return m.seekTo(m.positions[m.idx] + delta)
}

func (m *plotModel) seekTo(pos float64) tea.Cmd {
	if len(m.entries) == 0 {
		return nil
	}
	r := m.curRecipe()
	m.positions[m.idx] = math.Max(r.SS, math.Min(r.To, pos))
	if m.playing {
		m.halt()
		return m.play()
	}
	return nil
}

func waitForPlay(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done
		return playDoneMsg{}
	}
}

func (m plotModel) View() tea.View {
	var content string
	if len(m.entries) == 0 {
		content = "No slices to plot.\n"
	} else {
		e := m.entries[m.idx]
		r := e.recipe
		pos := m.positions[m.idx]
		content = fmt.Sprintf("[%d/%d] %s  %s / %s  [%s – %s]\n\n",
			m.idx+1, len(m.entries), e.label,
			fmtPos(pos-r.SS), audio.FmtDuration(r.To-r.SS),
			audio.FmtDuration(r.SS), audio.FmtDuration(r.To),
		)
		if m.loading || e.profile == nil {
			content += "  computing volume profile…\n"
		} else {
			// overhead: header(1) + blank(1) + chartLegend(1) + chartFooter(1) + blank(1) + summary(1) + blank(1) + nav(1)
			chartH := max(4, m.termH-8)
			content += audio.PlotProfile(e.profile, m.threshold, chartH)
			content += "\n" + profileSummary(e.profile, r.To, m.threshold)
		}
		content += "\n" + navHint(m.idx, len(m.entries), m.playing)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// profileSummary returns a one-line tail silence suggestion if detectable.
func profileSummary(profile []audio.ProfileWindow, sourceDuration, threshold float64) string {
	if len(profile) < 2 {
		return ""
	}
	lastLoudIdx := -1
	for i, w := range profile {
		if w.Volume.Mean > threshold {
			lastLoudIdx = i
		}
	}
	if lastLoudIdx < 0 || lastLoudIdx == len(profile)-1 {
		return ""
	}
	windowSecs := profile[1].SS - profile[0].SS
	silenceStart := profile[lastLoudIdx].SS + windowSecs
	return fmt.Sprintf("  suggested: --tail %.1f\n", sourceDuration-silenceStart)
}

// fmtPos formats seconds as M:SS.t (tenths) for sub-second position display.
func fmtPos(secs float64) string {
	total := int(secs * 10)
	tenths := total % 10
	s := (total / 10) % 60
	m := total / 600
	return fmt.Sprintf("%d:%02d.%d", m, s, tenths)
}

func navHint(idx, total int, playing bool) string {
	spaceAction := "play"
	if playing {
		spaceAction = "stop"
	}
	parts := []string{
		"q:quit",
		"space:" + spaceAction,
		"a:start  e:end  [:−10s  ]:+10s",
	}
	if idx > 0 {
		parts = append(parts, "←:prev")
	}
	if idx < total-1 {
		parts = append(parts, "→:next")
	}
	return strings.Join(parts, "  │  ")
}
