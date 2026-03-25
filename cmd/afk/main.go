package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	p := flag.String("p", "", "prompt to print")
	flag.Parse()
	if *p == "" {
		fmt.Fprintln(os.Stderr, "error: -p flag is required")
		os.Exit(2)
	}
	fmt.Println(*p)
}
