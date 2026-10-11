package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	inkMuted = "38;5;245"
	inkPink  = "38;5;218"
	inkWhite = "38;5;255"
	inkGreen = "38;5;115"
	inkAmber = "38;5;222"
	// Nerd Fonts: Material Design waveform and Font Awesome audio controls.
	iconWaveform   = "\U000f147d"
	iconHeadphones = "\uf025"
	iconClock      = "\uf017"
	iconVolume     = "\uf028"
	iconMute       = "\ueee8"
	iconPlay       = "\uf144"
	iconPause      = "\uf28b"
	iconStop       = "\uf28d"
)

type terminalUI struct {
	out         io.Writer
	file        *os.File
	ansi, color bool
	accent      string
	restore     func()
	live        bool
	paused      bool
	framed      bool
}

func newTerminalUI(out io.Writer) *terminalUI {
	u := &terminalUI{out: out, accent: inkPink, restore: func() {}}
	if file, ok := out.(*os.File); ok {
		u.file = file
		if os.Getenv("TERM") != "dumb" {
			u.ansi, u.restore = enableTerminalOutput(file)
		}
	}
	u.color = u.ansi && os.Getenv("NO_COLOR") == ""
	return u
}

func (u *terminalUI) paint(style, text string) string {
	if !u.color {
		return text
	}
	return "\x1b[" + style + "m" + text + "\x1b[0m"
}

func (u *terminalUI) width() int {
	if u.file != nil {
		if width, _, err := term.GetSize(int(u.file.Fd())); err == nil {
			return width
		}
	}
	return 80
}

func (u *terminalUI) heading(badge string) {
	fmt.Fprintf(u.out, "%s%s %s  %s\n", u.prefix("╭─"), u.paint(u.accent, iconWaveform), u.paint("1", "vibe noise"), u.paint(u.accent, badge))
}

// An open left border groups the four playback rows without adding padding
// rows or a right border that would clip long device names.
func (u *terminalUI) prefix(edge string) string {
	if !u.framed {
		return "  "
	}
	return "  " + u.paint(inkMuted, edge) + " "
}

func (u *terminalUI) key(label string) string {
	return u.paint("1;38;5;252;48;5;238", " "+label+" ")
}

func (u *terminalUI) playback(o options, names []string, keyboard bool) {
	u.framed = true
	if o.noise == "white" {
		u.accent = inkWhite
	}
	u.heading(fmt.Sprintf("%s · %g–%g Hz", strings.ToUpper(o.noise), o.low, o.high))
	session := "continuous"
	if o.duration > 0 {
		session = o.duration.String() + " timer"
	}
	outputs := fmt.Sprintf("%d outputs (all)", len(names))
	if o.device == "current" {
		outputs = "default output"
		if len(names) > 0 {
			outputs = "current: " + names[0]
		}
	}
	separator := u.paint(inkMuted, " · ")
	fmt.Fprintln(u.out, u.prefix("│ ")+u.paint(inkMuted, iconHeadphones)+" Stereo L=R"+separator+u.paint(inkMuted, iconClock)+" "+session+separator+u.paint(inkMuted, iconVolume)+" "+outputs)
	if o.verbose && o.device != "current" {
		for i, name := range names {
			branch := "├─"
			if i == len(names)-1 {
				branch = "└─"
			}
			fmt.Fprintln(u.out, u.prefix("│ ")+"  "+u.paint(inkMuted, branch+" "+name))
		}
	}
	if keyboard {
		u.shortcuts()
	} else {
		fmt.Fprintf(u.out, "%s%s %s\n", u.prefix("│ "), u.key("Ctrl+C"), u.paint(inkMuted, "stop · keyboard controls need a console"))
	}
	u.volume(o.volume)
}

func (u *terminalUI) shortcuts() {
	fmt.Fprintf(u.out, "%s%s %s %s %s %s %s %s %s\n", u.prefix("│ "),
		u.key("↑/↓"), u.paint(inkMuted, "±0.1dB"),
		u.key("Alt+↑/↓"), u.paint(inkMuted, "±1dB"),
		u.key("Space"), u.paint(inkMuted, "pause/resume"),
		u.key("Enter/Ctrl+C"), u.paint(inkMuted, "stop"))
}

func (u *terminalUI) volume(volume float64) {
	status, statusInk := iconPlay+" PLAYING", inkGreen
	if u.paused {
		status, statusInk = iconPause+" PAUSED", inkAmber
	}
	// Matching Nerd Font icons and padding keep the volume columns aligned.
	status = fmt.Sprintf("%-10s", status)
	badge := u.paint(statusInk, status)
	db := fmt.Sprintf("%+.1f dB", volumeDB(volume))
	volumeIcon := iconVolume
	if volume == 0 {
		db = "−∞ dB · muted"
		volumeIcon = iconMute
	}
	level := u.paint("1;"+u.accent, fmt.Sprintf("%6.2f%%", volume*100))
	line := u.prefix("╰─") + badge + " " + u.paint(inkMuted, volumeIcon) + " " + level + "  " + db
	// Reserve room for the status and the longest (muted) readout, leaving
	// one unused column so updating this line never wraps at the right edge.
	barWidth := min(20, u.width()-55)
	if barWidth >= 4 {
		halfSteps := int(math.Round(volume * float64(barWidth) * 2))
		filled := strings.Repeat("━", halfSteps/2)
		if halfSteps%2 != 0 {
			filled += "╸"
		}
		bar := u.paint(u.accent, filled) + u.paint(inkMuted, strings.Repeat("─", barWidth-(halfSteps+1)/2))
		line = u.prefix("╰─") + badge + " " + u.paint(inkMuted, volumeIcon) + " " + bar + " " + level + "  " + db
	}
	if u.ansi {
		fmt.Fprint(u.out, "\r\x1b[2K"+line)
		u.live = true
	} else {
		fmt.Fprintln(u.out, line)
	}
}

func (u *terminalUI) stopped(start time.Time, playbackErr error) {
	if u.live {
		fmt.Fprintln(u.out)
		u.live = false
	}
	status, color := "Stopped", inkGreen
	if playbackErr != nil {
		status, color = "Playback ended with an error", inkAmber
	}
	fmt.Fprintf(u.out, "  %s  %s\n", u.paint(color, iconStop+" "+status), u.paint(inkMuted, "after "+time.Since(start).Round(time.Second).String()))
}

func (u *terminalUI) devices(names []string) {
	u.heading("OUTPUT DEVICES")
	for i, name := range names {
		branch := "├─"
		if i == len(names)-1 {
			branch = "└─"
		}
		fmt.Fprintln(u.out, "  "+u.paint(inkMuted, branch)+" "+name)
	}
	fmt.Fprintf(u.out, "  %s\n", u.paint(inkMuted, "Use --device all or --device current"))
}

func printDiagnostic(label, message string) {
	u := newTerminalUI(os.Stderr)
	defer u.restore()
	fmt.Fprintf(u.out, "\n  %s  %s\n", u.paint(inkAmber, label), message)
}

func printHelp(out io.Writer) {
	u := newTerminalUI(out)
	defer u.restore()
	u.heading("PINK / WHITE NOISE")
	fmt.Fprintf(out, "  %s\n\n", u.paint("1", "vibe-noise [pink|p|white|w] [options]"))
	for _, row := range [][2]string{
		{"-v, --volume FLOAT", "Amplitude 0–1 · default 0.2"},
		{"-f, --freq LOW-HIGH", "Hz · pink 300–3000 / white 20–20000"},
		{"-d, --device MODE", "all or current · default all"},
		{"-t, --duration TIME", "30s, 5m, etc. · default until stopped"},
		{"-l, --list-devices", "Show output devices"},
		{"    --verbose", "Show device names during playback"},
		{"-h, --help", "Show this help"},
	} {
		fmt.Fprintf(out, "  %s %s\n", u.paint(u.accent, fmt.Sprintf("%-21s", row[0])), u.paint(inkMuted, row[1]))
	}
	fmt.Fprintln(out, "\n  Pink is the default. Frequency also accepts LOW~HIGH.")
	u.shortcuts()
	fmt.Fprintln(out, "  Gain is shown in dB; 0 dB = 100%.")
	fmt.Fprintf(out, "  %s\n", u.paint(inkMuted, "Try: vibe-noise pink -v 0.2 -f 300-3000 -d all"))
}
