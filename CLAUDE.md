# solar-radio

## Goal
Stream solar wind audio data from a remote URL and play it via mpv.

## Approach
3-line Go implementation:
1. `http.Get(url)` to open the stream
2. Pipe `resp.Body` directly to `cmd.Stdin`
3. Run mpv as a subprocess consuming stdin

```go
resp, err := http.Get(url)
cmd := exec.Command("mpv", "-")
cmd.Stdin = resp.Body
cmd.Run()
```

## Stations

### Synthwave / Electro
- https://stream.nightride.fm/nightride.mp3 — Nightride FM, 320kbps pure synthwave
- https://stream.nightride.fm/chillsynth.mp3 — chillwave synth
- https://stream.nightride.fm/darksynth.mp3 — darker edge

### Vaporwave
- https://radio.plaza.one/mp3 — Nightwave Plaza (vaporwave/city pop/future funk)

### YouTube
Pass a YouTube URL directly to mpv — it uses yt-dlp internally:
`mpv --no-video <youtube-url>`
