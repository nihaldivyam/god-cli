package main

import (
	"fmt"
	"god/cmd/alert" // <--- Import the new module
	"god/cmd/git"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "git":
		git.Handle(os.Args[2:])
	case "alert": // <--- Add this case
		alert.Handle(os.Args[2:])
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown module: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Usage: god <module> <command> [flags]")
	fmt.Println("\nAvailable Modules:")
	fmt.Println("  git    Manage git repositories")
	fmt.Println("  alert  Check Prometheus alerts") // <--- Update help text
	fmt.Println("\nExample:")
	fmt.Println("  god git pull --path=./work")
	fmt.Println("  god alert list")
}
