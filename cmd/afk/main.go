package main

import (
	"os"

	"github.com/jorgengundersen/afk/internal/afk"
)

func main() {
	os.Exit(afk.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
