package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/SurreptitiousFabric/omarchy-frame/internal/frame"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	result, err := frame.Run(args)
	if err != nil {
		_ = json.NewEncoder(stdout).Encode(map[string]any{"ok": false, "error": frame.PublicError(err)})
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
