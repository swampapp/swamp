package main

import (
	"fmt"
	"os"

	"github.com/swampapp/swamp/internal/credentials"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <repo-id>\f", os.Args[0])
		os.Exit(1)
	}
	repo := os.Args[1]
	fmt.Println(repo)
	os.Exit(0)
	settings := credentials.New(repo)
	settings.Delete()
}
