package mix

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/shmul/zfm/internal/audio"
)

type (
	previewPhase int

	plotEntry struct {
		label   string
		recipe  audio.Recipe
		profile []audio.ProfileWindow // nil until loaded
		volAdj  float64               // ephemeral user nudge in dB
	}

	plotModel struct {
		entries         []plotEntry
		positions       []float64
		idx             int
		threshold       float64
		previewSecs     float64
		previewPhase    previewPhase
		previewEntryIdx int
		previewTmpPath  string
		loading         bool
		playStartTime   time.Time
		playStartPos    float64
		stopPlay        func()
		playDone        <-chan struct{}
		termW           int
		termH           int
	}

	tickMsg struct{}

	profileLoadedMsg struct {
		idx     int
		profile []audio.ProfileWindow
	}

	playDoneMsg struct{}

	previewReadyMsg struct {
		phase    previewPhase
		entryIdx int
		tmpPath  string
		duration float64
		srcSS    float64 // cursor start in source-file coordinates
	}
)

const (
	phaseNone previewPhase = iota
	phaseHead
	phaseTail
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
	var positions []float64
	for i, s := range mf.Mix {
		if !shouldProcess(p.Just, i) {
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
		positions = append(positions, r.SS)
	}

	m := plotModel{entries: entries, positions: positions, threshold: p.SilenceThresh, previewSecs: p.Preview, termW: termW, termH: termH}
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

	case previewReadyMsg:
		if msg.tmpPath == "" || m.previewPhase == phaseNone {
			// encoding failed or was cancelled — discard and clean up
			if msg.tmpPath != "" {
				os.Remove(msg.tmpPath) //nolint:errcheck
			}
			return m, nil
		}
		m.previewTmpPath = msg.tmpPath
		m.previewPhase = msg.phase
		m.previewEntryIdx = msg.entryIdx
		m.idx = msg.entryIdx
		m.positions[msg.entryIdx] = msg.srcSS
		stop, done := audio.StartPlayAt(msg.tmpPath, 0, msg.duration)
		m.playStartTime = time.Now()
		m.playStartPos = msg.srcSS
		m.stopPlay = stop
		m.playDone = done
		return m, tea.Batch(waitForPlay(done), doTick())

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.haltAndWait()
			return m, tea.Quit

		case "space":
			if m.stopPlay != nil {
				m.halt()
				return m, nil
			}
			if m.previewSecs == 0 {
				return m, nil
			}
			// cancel in-flight encoding
			if m.previewPhase != phaseNone {
				m.previewPhase = phaseNone
				return m, nil
			}
			return m, m.play()

		case "+", "=":
			m.entries[m.idx].volAdj += 1.0
		case "-":
			m.entries[m.idx].volAdj -= 1.0

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
		if m.stopPlay != nil {
			m.updatePosition()
			return m, doTick()
		}

	case playDoneMsg:
		if m.stopPlay != nil {
			m.updatePosition()
			m.stopPlay = nil
			m.playDone = nil
		}
		if m.previewPhase != phaseNone {
			m.cleanupPreviewTemp()
			switch m.previewPhase {
			case phaseHead:
				m.previewPhase = phaseTail
				return m, m.encodePreview(m.previewEntryIdx, phaseTail)
			case phaseTail:
				next := m.previewEntryIdx + 1
				if next < len(m.entries) {
					m.previewEntryIdx = next
					m.previewPhase = phaseHead
					return m, m.encodePreview(next, phaseHead)
				}
				m.previewPhase = phaseNone
			}
		}
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
	m.stopPlay = nil
	m.playDone = nil
	m.cleanupPreviewTemp()
	m.previewPhase = phaseNone
}

func (m *plotModel) haltAndWait() {
	done := m.playDone
	m.halt()
	if done != nil {
		<-done
	}
}

func (m *plotModel) cleanupPreviewTemp() {
	if m.previewTmpPath != "" {
		os.Remove(m.previewTmpPath) //nolint:errcheck
		m.previewTmpPath = ""
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
	if m.previewSecs > 0 {
		m.previewEntryIdx = m.idx
		m.previewPhase = phaseHead
		return m.encodePreview(m.idx, phaseHead)
	}
	r := m.curRecipe()
	stop, done := audio.StartPlayAt(r.InputPath, m.positions[m.idx], r.To)
	m.playStartTime = time.Now()
	m.playStartPos = m.positions[m.idx]
	m.stopPlay = stop
	m.playDone = done
	return tea.Batch(waitForPlay(done), doTick())
}

// encodePreview renders a head or tail sub-recipe into a temp file and returns a previewReadyMsg.
// Runs in a goroutine (bubbletea cmd), so Execute is safe to call synchronously here.
func (m plotModel) encodePreview(idx int, phase previewPhase) tea.Cmd {
	r := m.entries[idx].recipe
	volAdj := m.entries[idx].volAdj
	previewSecs := m.previewSecs
	return func() tea.Msg {
		sub := r
		switch phase {
		case phaseHead:
			sub.To = math.Min(r.SS+previewSecs, r.To)
			sub.FadeOut = 0
			sub.Identical = false
		case phaseTail:
			sub.SS = math.Max(r.SS, r.To-previewSecs)
			sub.FadeIn = 0
			sub.Identical = false
		}
		sub.Volume = r.Volume + volAdj
		if volAdj != 0 {
			sub.Identical = false
		}
		ext := filepath.Ext(r.InputPath)
		tmp, err := os.CreateTemp("", "zfm-preview-*"+ext)
		if err != nil {
			return previewReadyMsg{phase: phase, entryIdx: idx}
		}
		tmp.Close()
		if err := audio.Execute(sub, tmp.Name()); err != nil {
			os.Remove(tmp.Name()) //nolint:errcheck
			return previewReadyMsg{phase: phase, entryIdx: idx}
		}
		return previewReadyMsg{
			phase:    phase,
			entryIdx: idx,
			tmpPath:  tmp.Name(),
			duration: sub.To - sub.SS,
			srcSS:    sub.SS,
		}
	}
}

func (m *plotModel) updatePosition() {
	elapsed := time.Since(m.playStartTime).Seconds()
	r := m.curRecipe()
	m.positions[m.idx] = math.Min(m.playStartPos+elapsed, r.To)
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
	if m.stopPlay != nil {
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

		phaseLabel := ""
		switch m.previewPhase {
		case phaseHead:
			phaseLabel = " [head]"
		case phaseTail:
			phaseLabel = " [tail]"
		}

		volStr := ""
		if e.volAdj != 0 {
			volStr = fmt.Sprintf("  vol: %+.1f dB", e.volAdj)
		}
		content = fmt.Sprintf("[%d/%d] %s%s  %s / %s  [%s – %s]%s\n\n",
			m.idx+1, len(m.entries), e.label, phaseLabel,
			fmtPos(pos-r.SS), audio.FmtDuration(r.To-r.SS),
			audio.FmtDuration(r.SS), audio.FmtDuration(r.To),
			volStr,
		)
		if m.loading || e.profile == nil {
			content += "  computing volume profile…\n"
		} else {
			// overhead: header(1) + blank(1) + chartLegend(1) + chartFooter(1) + blank(1) + summary(1) + blank(1) + nav(1)
			chartH := max(4, m.termH-8)
			content += audio.PlotProfile(e.profile, m.threshold, chartH)
			content += "\n" + profileSummary(e.profile, r.To, m.threshold)
		}
		content += "\n" + navHint(m.idx, len(m.entries), m.stopPlay != nil, m.previewSecs > 0, m.previewPhase)
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

func navHint(idx, total int, playing bool, previewMode bool, phase previewPhase) string {
	var spaceAction string
	switch {
	case playing:
		spaceAction = "stop"
	case previewMode && phase != phaseNone:
		spaceAction = "cancel"
	case previewMode:
		spaceAction = "preview"
	}
	parts := []string{"q:quit"}
	if spaceAction != "" {
		parts = append(parts, "space:"+spaceAction)
	}
	parts = append(parts, "a:start  e:end  [:−10s  ]:+10s  +/-:vol")
	if idx > 0 {
		parts = append(parts, "←:prev")
	}
	if idx < total-1 {
		parts = append(parts, "→:next")
	}
	return strings.Join(parts, "  │  ")
}
