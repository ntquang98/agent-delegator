package main

import (
	"fmt"
	"log"
	"os"

	"github.com/local/delegate-hub/bridge/internal/hub"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "serve" || os.Args[2] != "--stdio" {
		fmt.Fprintln(os.Stderr, "usage: delegate-hub serve --stdio")
		os.Exit(2)
	}
	manager, err := hub.NewManager(os.Getenv("DELEGATE_HUB_STATE_DIR"))
	if err != nil {
		log.Fatal(err)
	}
	if err := hub.NewServer(manager).Serve(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
