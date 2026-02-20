package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: minilink [options] <file1.o> [file2.o ...]")
		os.Exit(1)
	}
	fmt.Println("minilink: linker not yet implemented")
}
