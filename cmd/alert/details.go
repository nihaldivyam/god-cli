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
	filter := detailsCmd.String("filter", "", "Filter clusters by name (required)")
	namespace := detailsCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := detailsCmd.String("svc", "svc/alertmanager-operated", "Service name to port-forward")
	port := detailsCmd.String("port", "9093", "Local port to use")
	detailsCmd.Parse(args)

	if *filter == "" {
		fmt.Println("❌ Error: --filter is required for details (e.g., --filter staging2)")
		os.Exit(1)
	}

	// 1. TSH Discovery (Reusable logic)
	// We'll quickly re-implement the discovery to find the target cluster
	// (You could refactor this into a shared helper in scan.go later)
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

	// 2. Filter logic
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

	// 3. Iterate, Login, and Diagnose
	for _, cluster := range targetClusters {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("🌐 Connecting to: %s\n", cluster)

		// Login
		loginCmd := exec.Command("tsh", "kube", "login", cluster)
		if out, err := loginCmd.CombinedOutput(); err != nil {
			fmt.Printf("❌ Login failed: %v\n%s\n", err, string(out))
			continue
		}

		// Fetch Alerts
		fmt.Printf("🔌 Checking alerts...\n")
		alerts, err := FetchAlerts(*namespace, *service, *port)
		if err != nil {
			fmt.Printf("⚠️  Could not fetch alerts: %v\n", err)
			continue
		}

		if len(alerts) == 0 {
			fmt.Println("✅ No active alerts.")
			continue
		}

		fmt.Printf("🔥 Found %d active alerts:\n", len(alerts))

		// 4. Process Alerts & Run Rules
		processedRules := make(map[string]bool) // To avoid running same rule twice per cluster

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

			// CHECK RULES
			if ruleFunc, exists := DiagnosticRules[name]; exists {
				// Only run the diagnosis once per alert type per cluster
				// (e.g. don't run 'velero get backup' 10 times if there are 10 backup alerts)
				if !processedRules[name] {
					ruleFunc(alert)
					processedRules[name] = true
				}
			}
		}
	}
}
