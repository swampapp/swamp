package main

import (
	"fmt"
	"os"

	"github.com/swampapp/swamp/internal/resticsettings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <repo-id>\d", os.Args[0])
		os.Exit(1)
	}
	repo := os.Args[1]
	fmt.Println(repo)
	os.Exit(0)
	settings := resticsettings.New(repo)
	settings.Delete()
}
