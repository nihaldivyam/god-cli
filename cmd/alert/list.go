package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Alert represents the JSON structure returned by Alertmanager
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

func runList(args []string) {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	port := listCmd.String("port", "9093", "Local port (used for api proxy)")
	namespace := listCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := listCmd.String("svc", "svc/alertmanager-operated", "Service name")
	listCmd.Parse(args)

	clusterName := getClusterName()
	fmt.Printf("🌍 Cluster: %s\n", clusterName)

	alerts, err := FetchAlerts("", *namespace, *service, *port)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	printAlerts(alerts)
}

// FetchAlerts queries Alertmanager via the Kubernetes API server proxy
func FetchAlerts(server, namespace, service, port string) ([]Alert, error) {
	// Clean up service name (e.g., "svc/alertmanager-operated" -> "alertmanager-operated")
	svcName := strings.TrimPrefix(service, "svc/")

	// Construct the direct API proxy path to the Alertmanager pod
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%s/proxy/api/v2/alerts?active=true&inhibited=false&silenced=false&unprocessed=false", namespace, svcName, port)

	var output []byte
	var err error

	if server != "" {
		fmt.Println("   [SSH] Fetching alerts... (Touch YubiKey or enter sudo password if prompted)")

		cmdStr := fmt.Sprintf("sudo -i kubectl get --raw '%s'", apiPath)
		cmd := exec.Command("ssh", "-t", server, cmdStr)

		// Connect Stdin so the TTY can securely receive the YubiKey touch or password
		cmd.Stdin = os.Stdin
		output, err = cmd.CombinedOutput()
	} else {
		cmd := exec.Command("kubectl", "get", "--raw", apiPath)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch alerts: %v\n      Raw Output: %s", err, string(output))
	}

	// SSH with a TTY (-t) sometimes injects MOTD before the JSON,
	// and "Shared connection closed" after the JSON.
	// We scan the output to find the exact boundaries of the JSON array.
	outStr := string(output)
	startIdx := strings.Index(outStr, "[")
	endIdx := strings.LastIndex(outStr, "]")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return nil, fmt.Errorf("invalid response from alertmanager (expected JSON array): %s", outStr)
	}

	// Extract purely the JSON part (from the first '[' to the last ']')
	jsonBytes := []byte(outStr[startIdx : endIdx+1])

	var alerts []Alert
	if err := json.Unmarshal(jsonBytes, &alerts); err != nil {
		return nil, fmt.Errorf("failed to parse alerts JSON: %v\n      Extracted String: %s", err, string(jsonBytes))
	}

	return alerts, nil
}

func printAlerts(alerts []Alert) {
	if len(alerts) == 0 {
		fmt.Println("✅ No active alerts.")
		return
	}

	fmt.Printf("🔥 Found %d active alerts:\n", len(alerts))
	for _, alert := range alerts {
		name := alert.Labels["alertname"]
		ns := alert.Labels["namespace"]
		target := "cluster-wide"

		if pod, ok := alert.Labels["pod"]; ok {
			target = pod
		} else if instance, ok := alert.Labels["instance"]; ok {
			target = instance
		}
		if ns == "" {
			ns = "global"
		}

		fmt.Printf("   🔴 %-35s -> %s/%s\n", name, ns, target)
	}
	fmt.Println("")
}

func getClusterName() string {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}
