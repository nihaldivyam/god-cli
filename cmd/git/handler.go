// cmd/git/handler.go
package git

import (
	"fmt"
	"os"
)

// Handle processes the 'god git ...' commands
func Handle(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: god git <command>")
		fmt.Println("Commands: pull")
		os.Exit(1)
	}

	// Route based on the action
	switch args[0] {
	case "pull":
		// Pass the remaining flags to the pull command
		runPull(args[1:])
	default:
		fmt.Printf("Unknown git command: %s\n", args[0])
		os.Exit(1)
	}
}
