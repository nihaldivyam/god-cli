# god (Go Operations Daemon)

**god** 
is a blazing fast, extensible CLI tool designed to automate developer workflows. It features a robust bulk-git module for concurrent updates across hundreds of repos, and an alert module for instant multi-cluster Prometheus monitoring—all without hanging on network issues or password prompts

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

### 🐙 Git Module

Update all repositories in the current directory:
```bash
god git pull
```
### 🔔 Alert Module

Monitor Prometheus alerts across single or multiple Kubernetes clusters.

 - Zero-Config: Automatically handles kubectl port-forward to Alertmanager (and cleans it up).

 - Native Parsing: No need for jq; parses JSON output natively for speed.

| Command | Description |
| :--- | :--- |
| `god alert list` | Check alerts on the currently connected cluster (uses current kubectl context). |
| `god alert scan --filter <name>` | Scan all Teleport clusters matching the name. |

Flags:

 - --n: Namespace (default: monitoring)

 - --svc: Service name (default: svc/alertmanager-operated)

 - --port: Local port to use (default: 9093)

Example (Single Cluster):

```bash
god alert list
```

Example (Multi-Cluster Scan): This uses tsh to login to every cluster matching "cluster name" and checks for alerts.

```
god alert scan --filter prod
```

📂 Project Structure

```text
god/
├── go.mod
├── main.go            # CLI Entry Point (Router)
└── cmd/
    ├── git/           # Git Module
    │   ├── handler.go # Route handler
    │   └── pull.go    # Bulk git logic
    └── alert/         # Alert Module
        ├── handler.go # Route handler
        ├── list.go    # Single cluster logic
        └── scan.go    # Multi-cluster Teleport logic
```

🤝 Contributing
1. Fork the repository.

2. Create a feature branch (git checkout -b feature/amazing-feature).

3. Commit your changes (git commit -m 'Add some amazing feature').

4. Push to the branch (git push origin feature/amazing-feature).

5. Open a Pull Request.

📝 License
Distributed under the AGPL 3.0 License. See LICENSE for more information.
