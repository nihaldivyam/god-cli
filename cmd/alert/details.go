package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runDetails(args []string) {
	detailsCmd := flag.NewFlagSet("details", flag.ExitOnError)
	filter := detailsCmd.String("filter", "", "Filter clusters by name (Teleport)")
	server := detailsCmd.String("server", "", "Direct SSH connection string (e.g., ubuntu@192.12.3.1)")
	namespace := detailsCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := detailsCmd.String("svc", "svc/alertmanager-operated", "Service name to port-forward")
	port := detailsCmd.String("port", "9093", "Local port to use")
	detailsCmd.Parse(args)

	if *filter == "" && *server == "" {
		fmt.Println("❌ Error: Must provide either --filter (for Teleport) or --server (for SSH)")
		os.Exit(1)
	}

	// --- BRANCH 1: Direct SSH Server ---
	if *server != "" {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting via SSH to: %s\n", *server)

		fmt.Printf("🔌 Establishing SSH tunnel and checking alerts...\n")
		alerts, err := FetchAlerts(*server, *namespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			return
		}

		processAlerts(alerts, *server)
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

	fmt.Printf("🚀 Found %d clusters matching '%s'. Starting diagnosis...\n", len(targetClusters), *filter)

	for _, cluster := range targetClusters {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting to: %s\n", cluster)

		loginCmd := exec.Command("tsh", "kube", "login", cluster)
		if out, err := loginCmd.CombinedOutput(); err != nil {
			fmt.Printf("❌ Login failed: %v\n%s\n", err, string(out))
			continue
		}

		fmt.Printf("🔌 Checking alerts...\n")
		alerts, err := FetchAlerts("", *namespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			continue
		}

		processAlerts(alerts, "")
	}
}

// Helper to keep logic DRY
func processAlerts(alerts []Alert, server string) {
	if len(alerts) == 0 {
		fmt.Println("✅ No active alerts.")
		return
	}

	fmt.Printf("🔥 Found %d active alerts:\n", len(alerts))
	processedRules := make(map[string]bool)

	for _, alert := range alerts {
		name := alert.Labels["alertname"]
		ns := alert.Labels["namespace"]
		target := "cluster-wide"
		if pod, ok := alert.Labels["pod"]; ok {
			target = pod
		}
		if ns == "" {
			ns = "global"
		}

		fmt.Printf("   🔴 %-35s -> %s/%s\n", name, ns, target)

		if ruleFunc, exists := DiagnosticRules[name]; exists {
			if !processedRules[name] {
				ruleFunc(alert, server) // Pass the server context to the rules engine
				processedRules[name] = true
			}
		}
	}
}
