package alert

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Alert represents the JSON structure returned by amtool
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

func runList(args []string) {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	port := listCmd.String("port", "9093", "Local port to forward to")
	namespace := listCmd.String("n", "monitoring", "Namespace of the Alertmanager service")
	service := listCmd.String("svc", "svc/alertmanager-operated", "Service name to port-forward")
	listCmd.Parse(args)

	clusterName := getClusterName()
	fmt.Printf("🌍 Cluster: %s\n", clusterName)

	alerts, err := FetchAlerts(*namespace, *service, *port)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	printAlerts(alerts)
}

// FetchAlerts handles the port-forwarding and querying logic
func FetchAlerts(namespace, service, port string) ([]Alert, error) {
	// 1. Start Port-Forward
	// Note: We silence stdout to keep the CLI clean during multi-cluster scans
	pfCmd := exec.Command("kubectl", "port-forward", service, fmt.Sprintf("%s:%s", port, port), "-n", namespace)
	if err := pfCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if pfCmd.Process != nil {
			pfCmd.Process.Kill()
		}
	}()

	// 2. Wait for Port
	if !waitForPort("localhost", port, 5*time.Second) {
		return nil, fmt.Errorf("timed out waiting for port-forward")
	}

	// 3. Query amtool
	amURL := fmt.Sprintf("http://localhost:%s", port)
	cmd := exec.Command("amtool", "alert", "--alertmanager.url="+amURL, "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to query alertmanager: %v", err)
	}

	// 4. Parse
	var alerts []Alert
	if err := json.Unmarshal(output, &alerts); err != nil {
		return nil, fmt.Errorf("invalid json from amtool: %v", err)
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

// Helper: Get current k8s context
func getClusterName() string {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}

// Helper: Wait for TCP port
func waitForPort(host, port string, timeout time.Duration) bool {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return false
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
}
