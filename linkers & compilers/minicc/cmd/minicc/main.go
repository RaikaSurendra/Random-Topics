package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: minicc [options] <file.mc>")
		os.Exit(1)
	}
	fmt.Println("minicc: compiler not yet implemented")
	fmt.Printf("would compile: %s\n", os.Args[len(os.Args)-1])
}
