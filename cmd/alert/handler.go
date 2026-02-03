package alert

import (
	"fmt"
	"os"
)

// Handle processes 'god alert ...'
func Handle(args []string) {
	if len(args) < 1 {
		printHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		runList(args[1:])
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown alert command: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Usage: god alert <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  list    List active alerts (requires local port-forward)")
	fmt.Println("\nFlags:")
	fmt.Println("  --url   Alertmanager URL (default: http://localhost:9093)")
}
