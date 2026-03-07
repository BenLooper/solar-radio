package main

import (
	"fmt"
	"net/http"
	"os/exec"
)

const url = "https://stream.nightride.fm/nightride.mp3"

func main() {
	fmt.Println("Connecting to", url)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\nContent-Type: %s\n", resp.Status, resp.Header.Get("Content-Type"))

	cmd := exec.Command("mpv", "-")
	cmd.Stdin = resp.Body
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
