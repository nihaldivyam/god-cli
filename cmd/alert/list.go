package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

// Alert represents the JSON structure returned by amtool
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

func runList(args []string) {
	// 1. Parse Flags
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	amURL := listCmd.String("url", "http://localhost:9093", "Alertmanager URL")
	listCmd.Parse(args)

	// 2. Check if amtool is installed
	if _, err := exec.LookPath("amtool"); err != nil {
		fmt.Println("❌ Error: 'amtool' is not installed or not in PATH.")
		fmt.Println("Please install it first: go install github.com/prometheus/alertmanager/cmd/amtool@latest")
		os.Exit(1)
	}

	// 3. Execute amtool command
	// We ask for JSON output so Go can parse it reliably
	cmd := exec.Command("amtool", "alert", "--alertmanager.url="+*amURL, "-o", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Failed to contact Alertmanager at %s\n", *amURL)
		fmt.Println("---------------------------------------------------")
		fmt.Println("💡 Tip: Did you run the port-forward?")
		fmt.Println("   kubectl port-forward svc/alertmanager-operated 9093:9093 -n monitoring")
		fmt.Println("---------------------------------------------------")
		fmt.Printf("Error details: %v\n", err)
		os.Exit(1)
	}

	// 4. Parse JSON
	var alerts []Alert
	if err := json.Unmarshal(output, &alerts); err != nil {
		fmt.Printf("❌ Error parsing amtool output: %v\n", err)
		os.Exit(1)
	}

	// 5. Display Results
	if len(alerts) == 0 {
		fmt.Println("✅ No active alerts found.")
		return
	}

	fmt.Printf("🔥 Found %d active alerts:\n\n", len(alerts))

	for _, alert := range alerts {
		name := alert.Labels["alertname"]
		namespace := alert.Labels["namespace"]

		// Determine the target (Pod > Instance > Cluster-wide)
		target := "cluster-wide"
		if pod, ok := alert.Labels["pod"]; ok {
			target = pod
		} else if instance, ok := alert.Labels["instance"]; ok {
			target = instance
		}

		// Handle missing namespace
		if namespace == "" {
			namespace = "global"
		}

		// Print formatted output: AlertName -> Namespace/Target
		fmt.Printf("🔴 %-30s -> %s/%s\n", name, namespace, target)
	}
}
