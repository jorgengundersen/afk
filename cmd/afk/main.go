package main

import (
	"fmt"
	"os"

	"github.com/jorgengundersen/afk/internal/config"
)

func main() {
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	fmt.Println(cfg.Prompt)
}
