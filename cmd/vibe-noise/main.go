package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const sampleRate = 44100

type options struct {
	noise, device     string
	volume, low, high float64
	duration          time.Duration
	list              bool
	verbose           bool
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var o options
	var frequency string
	o.noise = "pink"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.noise, args = strings.ToLower(args[0]), args[1:]
	}
	switch o.noise {
	case "p":
		o.noise = "pink"
	case "w":
		o.noise = "white"
	}
	fs := flag.NewFlagSet("vibe-noise", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.Float64Var(&o.volume, "volume", 0.2, "output amplitude from 0 (silent) to 1")
	fs.StringVar(&frequency, "freq", "", "frequency range in Hz, e.g. 300-3000 or 300~3000 (pink: 300-3000; white: 20-20000)")
	fs.StringVar(&o.device, "device", "all", "output devices: all or current (Windows default output)")
	fs.DurationVar(&o.duration, "duration", 0, "stop after this duration, e.g. 30s or 5m (0: until Enter or Ctrl+C)")
	fs.BoolVar(&o.list, "list-devices", false, "list output devices and exit")
	fs.BoolVar(&o.verbose, "verbose", false, "show output device names during playback")
	fs.Usage = func() {
		printHelp(output)
	}
	var err error
	args, err = normalizeFlags(args, fs)
	if err != nil {
		return o, err
	}
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 {
		return o, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	o.noise = strings.ToLower(o.noise)
	o.device = strings.ToLower(o.device)
	if o.noise != "pink" && o.noise != "white" {
		return o, errors.New("noise type must be pink, p, white, or w")
	}
	if o.device != "all" && o.device != "current" {
		return o, errors.New("--device must be all or current")
	}
	if !finite(o.volume) || o.volume < 0 || o.volume > 1 {
		return o, errors.New("--volume must be between 0 and 1")
	}
	if o.duration < 0 {
		return o, errors.New("--duration must be nonnegative")
	}
	o.low, o.high = 300, 3000
	if o.noise == "white" {
		o.low, o.high = 20, 20000
	}
	if frequency != "" {
		parts := strings.Split(strings.ReplaceAll(frequency, "~", "-"), "-")
		if len(parts) != 2 {
			return o, errors.New("--freq must be a range in Hz, e.g. 300-3000")
		}
		var err error
		o.low, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return o, fmt.Errorf("invalid lower frequency: %w", err)
		}
		o.high, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return o, fmt.Errorf("invalid upper frequency: %w", err)
		}
	}
	if !finite(o.low) || !finite(o.high) || o.low <= 0 || o.low >= o.high || o.high >= sampleRate/2 {
		return o, fmt.Errorf("--freq must satisfy 0 < low < high < %d Hz", sampleRate/2)
	}
	return o, nil
}

// flag accepts single-dash long names by default. Enforce the CLI convention
// here, while leaving values (including negative numbers) untouched.
func normalizeFlags(args []string, fs *flag.FlagSet) ([]string, error) {
	short := map[string]string{"v": "volume", "f": "freq", "d": "device", "t": "duration", "l": "list-devices", "h": "help"}
	var result []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(result, args[i:]...), nil
		}
		if !strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unexpected argument %q; noise type must precede options", arg)
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "-"), "=")
		if strings.HasPrefix(name, "-") {
			name = strings.TrimPrefix(name, "-")
			if len(name) <= 1 {
				return nil, fmt.Errorf("use a single dash for short option %q", arg)
			}
		} else {
			long, ok := short[name]
			if !ok {
				return nil, fmt.Errorf("unknown short option %q; full option names require --", arg)
			}
			name = long
		}
		f := fs.Lookup(name)
		if f == nil && name != "help" {
			return nil, fmt.Errorf("unknown option --%s", name)
		}
		normalized := "--" + name
		if hasValue {
			normalized += "=" + value
		}
		result = append(result, normalized)
		if !hasValue && name != "help" && name != "list-devices" && name != "verbose" && i+1 < len(args) {
			i++
			result = append(result, args[i])
		}
	}
	return result, nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func run(args []string) error {
	o, err := parseOptions(args, os.Stderr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return play(ctx, o)
}

func main() {
	if err := run(os.Args[1:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		printDiagnostic("ERROR", err.Error())
		os.Exit(1)
	}
}
