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
