package main

import (
	"flag"
	"fmt"
)

func main() {
	p := flag.String("p", "", "prompt to print")
	flag.Parse()
	fmt.Println(*p)
}
