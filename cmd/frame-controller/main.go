package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/swa/omarchy-frame/internal/frame"
)

func main() {
	result, err := frame.Run(os.Args[1:])
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
