// cmd/git/pull.go
package git

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func runPull(args []string) {
	// Define flags specific to this command
	pullCmd := flag.NewFlagSet("pull", flag.ExitOnError)
	workDir := pullCmd.String("path", ".", "Target directory containing git repositories")
	dryRun := pullCmd.Bool("dry-run", false, "Check for updates without modifying files")
	verbose := pullCmd.Bool("v", false, "Show detailed git output")

	pullCmd.Parse(args)

	rootPath, _ := filepath.Abs(*workDir)

	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		fmt.Printf("Error: Directory '%s' does not exist.\n", rootPath)
		os.Exit(1)
	}

	fmt.Printf("🚀 Scanning %s for git repositories...\n", rootPath)

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	repoCount := 0
	start := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(rootPath, entry.Name())
			if isGitRepo(fullPath) {
				repoCount++
				wg.Add(1)
				go func(path, name string) {
					defer wg.Done()
					processRepo(path, name, *dryRun, *verbose)
				}(fullPath, entry.Name())
			}
		}
	}

	wg.Wait()
	fmt.Printf("\n--- Processed %d repositories in %s ---\n", repoCount, time.Since(start).Round(time.Millisecond))
}

// Helper functions (private to the git package)

func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info.IsDir()
}

func processRepo(path, name string, dryRun, verbose bool) {
	var cmd *exec.Cmd
	if dryRun {
		cmd = exec.Command("git", "fetch", "--dry-run")
	} else {
		cmd = exec.Command("git", "pull")
	}

	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		fmt.Printf("❌ [%s] Error\n", name)
		if verbose {
			fmt.Printf("\t%s\n", strings.ReplaceAll(outputStr, "\n", "\n\t"))
		}
		return
	}

	if dryRun {
		if len(outputStr) > 0 {
			fmt.Printf("🔄 [%s] Updates available (Dry Run)\n", name)
		} else {
			fmt.Printf("✅ [%s] Up to date (Dry Run)\n", name)
		}
	} else {
		if strings.Contains(outputStr, "Already up to date") {
			fmt.Printf("✅ [%s] Up to date\n", name)
		} else {
			fmt.Printf("⬇️  [%s] Updated\n", name)
			if verbose {
				fmt.Printf("\t%s\n", strings.ReplaceAll(outputStr, "\n", "\n\t"))
			}
		}
	}
}
