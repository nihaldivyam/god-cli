package alert

import (
	"fmt"
	"os"
)

func Handle(args []string) {
	if len(args) < 1 {
		printHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		runList(args[1:])
	case "scan":
		runScan(args[1:])
	case "details":
		runDetails(args[1:]) // <--- Add this
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
	fmt.Println("  list     List alerts on current cluster")
	fmt.Println("  scan     Scan multiple Teleport clusters")
	fmt.Println("  details  Scan and run diagnostics on matching alerts")
	fmt.Println("\nFlags:")
	fmt.Println("  --filter <name>   Filter clusters (required for details)")
}
