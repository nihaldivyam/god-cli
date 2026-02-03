package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TSHCluster represents the JSON output from `tsh kube ls --format=json`
// Updated to match your specific Teleport version output.
type TSHCluster struct {
	Name   string            `json:"kube_cluster_name"` // The fix: matches your JSON key
	Labels map[string]string `json:"labels"`
}

func runScan(args []string) {
	scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
	filter := scanCmd.String("filter", "", "Filter clusters by name (e.g., 'kilroy')")
	namespace := scanCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := scanCmd.String("svc", "svc/alertmanager-operated", "Service name to port-forward")
	port := scanCmd.String("port", "9093", "Local port to use")
	scanCmd.Parse(args)

	// 1. Check for tsh
	if _, err := exec.LookPath("tsh"); err != nil {
		fmt.Println("❌ Error: 'tsh' (Teleport) is not installed.")
		os.Exit(1)
	}

	// 2. Get list of clusters from Teleport
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

	// 3. Filter Clusters
	var targetClusters []string
	for _, c := range allClusters {
		// Filter based on the Name field
		if *filter == "" || strings.Contains(c.Name, *filter) {
			targetClusters = append(targetClusters, c.Name)
		}
	}

	if len(targetClusters) == 0 {
		fmt.Printf("⚠️  No clusters found matching '%s'\n", *filter)
		return
	}

	fmt.Printf("🚀 Found %d clusters matching '%s'. Starting scan...\n\n", len(targetClusters), *filter)

	// 4. Iterate and Scan
	for _, cluster := range targetClusters {
		fmt.Printf("--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting to: %s\n", cluster)

		// Login to the cluster
		loginCmd := exec.Command("tsh", "kube", "login", cluster)
		if out, err := loginCmd.CombinedOutput(); err != nil {
			fmt.Printf("❌ Login failed: %v\n%s\n", err, string(out))
			continue
		}

		// Reuse the fetch logic from list.go
		fmt.Printf("🔌 Checking alerts...\n")
		alerts, err := FetchAlerts(*namespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			continue
		}

		printAlerts(alerts)
	}
}
