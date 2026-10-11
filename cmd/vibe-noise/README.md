# vibe-noise

Continuous pink or white noise for Windows, written in Go. Uses WinMM with no
CGO or additional dependencies. Both stereo channels contain exactly the same
samples.

The four-row playback display uses a slim Unicode border, Nerd Font audio icons,
highlighted shortcut keys, and a live volume bar with pink/white accents.
PLAYING/PAUSED labels occupy a fixed width. Narrow terminals use a compact volume readout. Redirected
output contains no ANSI escape codes; set `NO_COLOR=1` to disable colors.
Output device names are hidden by default; use `--verbose` to show the list.
With `--device current`, the selected device name appears on the summary line.

```powershell
go run ./cmd/vibe-noise
go run ./cmd/vibe-noise pink --volume 0.2 --freq 300-3000 --device all
go run ./cmd/vibe-noise w -v 0.1 -f 200-8000 -d current
go run ./cmd/vibe-noise p --freq "300~3000" --duration 5m
go run ./cmd/vibe-noise --list-devices
go build -o vibe-noise.exe ./cmd/vibe-noise
./vibe-noise.exe w -v 0.15 --device current
```

Usage: `vibe-noise [pink|p|white|w] [options]`. Noise type comes before options
and defaults to `pink`. Use `--` for full option names and `-` for single-letter
options. Both `--volume=0.2` and `-v 0.2` work; `-volume` is rejected.

| Option | Short | Default | Meaning |
| --- | --- | --- | --- |
| `--volume` | `-v` | `0.2` | Amplitude multiplier, 0–1; 0 is silent |
| `--freq` | `-f` | Pink: `300-3000`; white: `20-20000` | Frequency range in Hz; `LOW-HIGH` or `LOW~HIGH` |
| `--device` | `-d` | `all` | All available output devices, or `current` for the Windows default output |
| `--duration` | `-t` | `0` | Duration such as `30s` or `5m`; 0 runs until Enter or Ctrl+C |
| `--list-devices` | `-l` | | List output devices and exit |
| `--verbose` | | `false` | Show output device names during playback |
| `--help` | `-h` | | Show help |

Audio uses 44.1 kHz, 16-bit stereo. Frequency limits must satisfy
`0 < LOW < HIGH < 22050`. Fourth-order Butterworth high-pass and low-pass filters
provide smooth rolloffs (approximately −3 dB at each edge for well-separated
edges), rather than hard frequency cutoffs. Pink noise uses
[Paul Kellet's refined filter](https://www.firstpr.com.au/dsp/pink-noise/).
Volume scales this application's samples, leaving system volume unchanged;
perceived loudness also depends on the selected range and output device.

During playback, **Enter** or **Ctrl+C** stops. **Up/Down arrows** raise/lower
volume on all selected devices by **0.1 dB** per press; holding a key repeats.
**Alt+Up/Down** adjusts by **1 dB**. **Space** pauses/resumes all outputs without
resetting the noise generator. If the volume changes while paused, stale queued
audio is discarded and refilled at the new level before resuming. Holding Space
toggles once until you release it.
The PLAYING/PAUSED status appears beside the volume. A duration timer continues
counting while paused.
The live display shows percentage and gain in dB, calculated as
`20 * log10(volume)` (20% is −14.0 dB; 100% is 0 dB). This is application gain,
not measured sound pressure or signal RMS. The maximum is 0 dB; lowering below
−80 dB mutes (`-Inf dB`), and Up from mute starts at −80 dB. Already queued audio
can take about 200 ms to reflect a change. Keyboard controls require an
interactive Windows console; redirected stdin leaves Ctrl+C and duration active.

`all` opens each output exposed by WinMM at startup; unavailable devices are
reported and skipped. The command fails if none can be opened. `current`
selects the default output at startup. Restart to pick up device/default changes.
Device names come from Core Audio to avoid WinMM's 31-character truncation,
with legacy names as a fallback when the driver cannot supply an endpoint ID.
All outputs receive the same noise sequence, but separate hardware clocks and
output latency mean devices are not synchronized with each other.

Buffers remain pinned while Windows uses them and are released after stopping
and unpreparing them, following the
[WinMM buffer lifecycle](https://learn.microsoft.com/en-us/windows/win32/multimedia/audio-data-blocks).
Playback is currently Windows-only.

```powershell
go test ./cmd/vibe-noise
go vet ./cmd/vibe-noise
```
