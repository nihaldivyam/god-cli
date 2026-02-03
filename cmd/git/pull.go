package git

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// maxConcurrency limits how many git processes run at once to prevent network choking
const maxConcurrency = 10

func runPull(args []string) {
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

	// Create a buffered channel to act as a semaphore (limit concurrency)
	sem := make(chan struct{}, maxConcurrency)

	start := time.Now()

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(rootPath, entry.Name())
			if isGitRepo(fullPath) {
				repoCount++
				wg.Add(1)

				// Acquire a slot in the semaphore
				sem <- struct{}{}

				go func(path, name string) {
					defer wg.Done()
					defer func() { <-sem }() // Release the slot when done
					processRepo(path, name, *dryRun, *verbose)
				}(fullPath, entry.Name())
			}
		}
	}

	wg.Wait()
	fmt.Printf("\n--- Processed %d repositories in %s ---\n", repoCount, time.Since(start).Round(time.Millisecond))
}

func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info.IsDir()
}

func processRepo(path, name string, dryRun, verbose bool) {
	// 1. Strict Timeout: 15 seconds per repo is plenty.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	if dryRun {
		cmd = exec.CommandContext(ctx, "git", "fetch", "--dry-run")
	} else {
		cmd = exec.CommandContext(ctx, "git", "pull")
	}

	cmd.Dir = path

	// 2. Network & Auth hardening
	env := os.Environ()
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	// Added ConnectTimeout=5 to fail fast on bad connections
	env = append(env, "GIT_SSH_COMMAND=ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=accept-new")
	cmd.Env = env

	if verbose {
		fmt.Printf("👉 [%s] Checking...\n", name)
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 3. Timeout Handling
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Printf("⏳ [%s] Timed out (Network stuck)\n", name)
		return
	}

	// 4. Error Handling
	if err != nil {
		if strings.Contains(outputStr, "terminal prompts disabled") ||
			strings.Contains(outputStr, "Authentication failed") ||
			strings.Contains(outputStr, "Permission denied") {

			fmt.Printf("🔒 [%s] Skipped (Auth required)\n", name)
			return
		}

		// Clean up the output for display
		cleanOutput := strings.TrimSpace(outputStr)
		if len(cleanOutput) > 0 {
			// If verbose, show the full error, otherwise just a summary
			if verbose {
				fmt.Printf("❌ [%s] Error: %v\n\t%s\n", name, err, strings.ReplaceAll(cleanOutput, "\n", "\n\t"))
			} else {
				// Often the first line of git error is enough
				firstLine := strings.Split(cleanOutput, "\n")[0]
				fmt.Printf("❌ [%s] Failed: %s\n", name, firstLine)
			}
		} else {
			fmt.Printf("❌ [%s] Failed: %v\n", name, err)
		}
		return
	}

	// 5. Success Handling
	if dryRun {
		// git fetch --dry-run produces output only if there are updates (usually)
		// However, sometimes it is silent if up to date.
		if len(outputStr) > 0 {
			fmt.Printf("🔄 [%s] Updates available\n", name)
		} else {
			fmt.Printf("✅ [%s] Up to date\n", name)
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
