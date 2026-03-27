package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TSHCluster represents the JSON output from `tsh kube ls`
type TSHCluster struct {
	Name string `json:"kube_cluster_name"` // <--- Fix is here
}

func runScan(args []string) {
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	filter := scanCmd.String("filter", "", "Filter clusters by name (Teleport)")
	server := scanCmd.String("server", "", "Direct SSH connection string (e.g., ubuntu@192.12.3.1)")
	namespace := scanCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := scanCmd.String("svc", "svc/alertmanager-operated", "Service name")
	port := scanCmd.String("port", "9093", "Local port to use")
	scanCmd.Parse(args)

	if *filter == "" && *server == "" {
		fmt.Println("❌ Error: Must provide either --filter (for Teleport) or --server (for SSH)")
		os.Exit(1)
	}

	// --- Smart Namespace Override ---
	actualNamespace := *namespace
	if *server != "" && actualNamespace == "monitoring" {
		actualNamespace = "monitoring-linuxaid"
	}

	// --- BRANCH 1: Direct SSH Server ---
	if *server != "" {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting via SSH to: %s\n", *server)

		// Use the dynamically selected namespace here
		alerts, err := FetchAlerts(*server, actualNamespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			return
		}

		printAlerts(alerts)
		return
	}

	// --- BRANCH 2: Teleport Discovery ---
	if _, err := exec.LookPath("tsh"); err != nil {
		fmt.Println("❌ Error: 'tsh' is not installed.")
		os.Exit(1)
	}

	fmt.Println("🔍 Fetching clusters from Teleport...")
	cmd := exec.Command("tsh", "kube", "ls", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Failed to run 'tsh kube ls': %v\n", err)
		os.Exit(1)
	}

	var allClusters []TSHCluster
	if err := json.Unmarshal(output, &allClusters); err != nil {
		fmt.Printf("❌ Failed to parse tsh output: %v\n", err)
		os.Exit(1)
	}

	var targetClusters []string
	for _, c := range allClusters {
		if strings.Contains(c.Name, *filter) {
			targetClusters = append(targetClusters, c.Name)
		}
	}

	if len(targetClusters) == 0 {
		fmt.Printf("⚠️  No clusters found matching '%s'\n", *filter)
		return
	}

	fmt.Printf("🚀 Found %d clusters matching '%s'. Starting scan...\n", len(targetClusters), *filter)

	for _, cluster := range targetClusters {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting to: %s\n", cluster)

		loginCmd := exec.Command("tsh", "kube", "login", cluster)
		if out, err := loginCmd.CombinedOutput(); err != nil {
			fmt.Printf("❌ Login failed: %v\n%s\n", err, string(out))
			continue
		}

		fmt.Printf("🔌 Checking alerts...\n")
		// Use the dynamically selected namespace here as well
		alerts, err := FetchAlerts("", actualNamespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			continue
		}

		printAlerts(alerts)
	}
}
