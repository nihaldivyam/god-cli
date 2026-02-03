# god (Go Ops Daemon)

**god** is a blazing fast, extensible CLI tool designed to automate developer workflows across multiple repositories. 

It currently features a robust bulk-git module that allows you to pull, fetch, or inspect updates across hundreds of local repositories concurrently without hanging on network issues or password prompts.

## 🚀 Features

* **⚡ Concurrent Execution:** Updates multiple repositories in parallel using Go routines, limited by a worker pool to prevent network throttling.
* **🛡️ Hang-Proof:** Built-in timeouts (15s) and SSH strict modes ensure the tool never gets stuck on bad networks or unresponsive servers.
* **🔐 Auth-Aware:** Automatically detects and skips repositories requiring manual password input, preventing the terminal from freezing.
* **🧪 Dry Run Mode:** Preview updates (`git fetch`) without modifying your local files.
* **📂 Extensible Architecture:** Structured cleanly to easily add new modules (e.g., `docker`, `file`, `clean`) in the future.

## 📦 Installation

### Prerequisites
* [Go](https://go.dev/dl/) 1.20 or higher
* Git installed and available in your PATH

### Build from Source

1.  Clone this repository:
    ```bash
    git clone [https://github.com/yourusername/god.git](https://github.com/yourusername/god.git)
    cd god
    ```

2.  Build the binary:
    ```bash
    go build -o god main.go
    ```

3.  Move to your PATH (Linux/macOS):
    ```bash
    sudo mv god /usr/local/bin/
    ```

    *(For Windows, move `god.exe` to a folder in your PATH)*

## 🛠️ Usage

The basic syntax is `god <module> <command> [flags]`.

example `god git pull --path /Users/work/repos/`

### Git Module

Update all repositories in the current directory:
```bash
god git pull
