package main

import (
	"fmt"
	"os"
)

func runMake(args []string) {
	state := readState()
	err := generateServerConfig(state, os.Stdout)
	if err != nil {
		fmt.Println("Error: error when generating config:", err)
	}
}
