package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	cropCmd struct {
		Start          string  `short:"s" long:"start" description:"start position ([hh:]mm:ss)"`
		End            string  `short:"e" long:"end" description:"end position ([hh:]mm:ss)"`
		Head           float64 `short:"h" long:"head" default:"0" description:"head offset (secs)"`
		Tail           float64 `short:"t" long:"tail" default:"0" description:"tail offset from end (secs)"`
		FadeIn         float64 `short:"i" long:"fade-in" default:"0" description:"fade in duration (secs)"`
		FadeOut        float64 `short:"o" long:"fade-out" default:"0" description:"fade out duration (secs)"`
		FadeCurve      string  `long:"fade-curve" default:"qsin" description:"afade curve (qsin, tri, log, ...)"`
		Play           bool    `short:"p" long:"play" description:"play instead of save"`
		TargetDir      string  `long:"target-dir" description:"output directory"`
		DryRun         bool    `short:"n" long:"dry-run" description:"prepare but don't write"`
		AnalyzeSilence bool    `short:"a" long:"analyze-silence" description:"analyze tail for silence"`
		SilenceThresh  float64 `long:"silence-thresh" default:"-40.0" description:"silence threshold (dBFS)"`
		Args           struct {
			Filename string `positional-arg-name:"filename" required:"true"`
		} `positional-args:"true"`
	}

	playlistCmd struct {
		TargetDir     string  `long:"target-dir" description:"output directory"`
		Preview       float64 `short:"p" long:"preview" default:"0" description:"play preview (secs) of head and tail"`
		Just          int     `short:"j" long:"just" default:"-1" description:"process only track N (zero-based)"`
		OneByOne      bool    `short:"1" long:"one-by-one" description:"play tracks individually"`
		DryRun        bool    `short:"n" long:"dry-run" description:"prepare but don't write"`
		SilenceThresh float64 `long:"silence-thresh" default:"-50.0" description:"silence threshold for profile (dBFS)"`
		Args          struct {
			Filename string `positional-arg-name:"filename" required:"true"`
		} `positional-args:"true"`
	}

	m3uCmd struct {
		TargetDir string `long:"target-dir" description:"output directory for csv file"`
		Args      struct {
			Filename string `positional-arg-name:"filename" required:"true"`
		} `positional-args:"true"`
	}

	playCmd struct {
		Head      float64 `short:"h" long:"head" default:"0" description:"head offset (secs)"`
		Tail      float64 `short:"t" long:"tail" default:"0" description:"tail offset from end (secs)"`
		FadeIn    float64 `short:"i" long:"fade-in" default:"0" description:"fade in (secs)"`
		FadeOut   float64 `short:"o" long:"fade-out" default:"0" description:"fade out (secs)"`
		FadeCurve string  `long:"fade-curve" default:"qsin" description:"afade curve (qsin, tri, log, ...)"`
		Args      struct {
			Files []string `positional-arg-name:"files" required:"true"`
		} `positional-args:"true"`
	}

	generateCmd struct {
		Args struct {
			Dir string `positional-arg-name:"dir" required:"true"`
		} `positional-args:"true"`
	}

	analyzeCmd struct {
		AnalysisDuration float64 `short:"d" long:"analysis-duration" default:"30.0" description:"seconds of tail to analyze"`
		SilenceThresh    float64 `short:"t" long:"silence-thresh" default:"-40.0" description:"silence threshold (dBFS)"`
		MinSilence       float64 `short:"m" long:"min-silence" default:"1.0" description:"minimum silence duration (secs)"`
		Args             struct {
			Filename string `positional-arg-name:"filename" required:"true"`
		} `positional-args:"true"`
	}

	mixCmd struct {
		TargetDir     string  `long:"target-dir" description:"output directory"`
		Preview       float64 `short:"p" long:"preview" default:"0" description:"play preview (secs) of head and tail"`
		Just          int     `short:"j" long:"just" default:"-1" description:"process only slice N (zero-based)"`
		DryRun        bool    `short:"n" long:"dry-run" description:"prepare but don't write"`
		SilenceThresh float64 `long:"silence-thresh" default:"-50.0" description:"silence threshold for profile (dBFS)"`
		Args          struct {
			Filename string `positional-arg-name:"filename" required:"true"`
		} `positional-args:"true"`
	}
)

type zfmOpts struct {
	Verbose  bool        `short:"v" long:"verbose" description:"print ffmpeg invocation lines"`
	Crop     cropCmd     `command:"crop" description:"Crop audio file"`
	Playlist playlistCmd `command:"playlist" description:"Batch process from CSV"`
	M3U      m3uCmd      `command:"m3u" description:"Convert M3U playlist to CSV"`
	Play     playCmd     `command:"play" description:"Play files"`
	Generate generateCmd `command:"generate" description:"Generate playlist.csv from directory"`
	Analyze  analyzeCmd  `command:"analyze" description:"Analyze audio for silence"`
	Mix      mixCmd      `command:"mix" description:"Compose from .mix file"`
}

var o zfmOpts

func verbose() {
	if o.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	parser := flags.NewParser(&o, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
