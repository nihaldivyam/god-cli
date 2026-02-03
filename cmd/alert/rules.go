package alert

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
)

// RuleFunc defines the function signature for a diagnostic check
type RuleFunc func(alert Alert)

// DiagnosticRules maps an "AlertName" to a specific function
var DiagnosticRules = map[string]RuleFunc{
	"VeleroUnsuccessfulBackup": checkVeleroBackup,
	"ArgoCdAppUnhealthy":       checkArgoUnhealthy,
}

// --- Prometheus Response Structs ---

type PromResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// --- Rule Implementations ---

func checkArgoUnhealthy(alert Alert) {
	fmt.Println("\n   🔍 [Diagnosis] Querying Prometheus for unhealthy ArgoCD apps...")

	// 1. Construct the Query
	promQuery := `argocd_app_info{health_status!="Healthy"}`

	apiPath := fmt.Sprintf("/api/v1/namespaces/monitoring/services/prometheus-operated:9090/proxy/api/v1/query?query=%s", url.QueryEscape(promQuery))

	cmd := exec.Command("kubectl", "get", "--raw", apiPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("      ❌ Failed to query Prometheus: %v\n", err)
		return
	}

	var resp PromResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		fmt.Printf("      ❌ Failed to parse Prometheus JSON: %v\n", err)
		return
	}

	if len(resp.Data.Result) == 0 {
		fmt.Println("      ✅ Prometheus returned no unhealthy apps")
		return
	}

	// 2. Setup Tabwriter for perfect alignment
	// minwidth, tabwidth, padding, padchar, flags
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	for _, res := range resp.Data.Result {
		appName := res.Metric["name"]
		health := res.Metric["health_status"]
		sync := res.Metric["sync_status"]
		destNs := res.Metric["dest_namespace"]

		// We use \t (tab) to separate columns
		fmt.Fprintf(w, "      ⚠️  App: %s\t| Health: %s\t| Sync: %s\t| Ns: %s\n",
			appName, health, sync, destNs)
	}

	// Flush the buffer to print aligned output
	w.Flush()
	fmt.Println("")
}

func checkVeleroBackup(alert Alert) {
	fmt.Println("\n   🔍 [Diagnosis] Running: velero get backup (showing top 5)")

	cmd := exec.Command("velero", "get", "backup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("      ❌ Failed to run velero: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	limit := 5
	if len(lines) < limit {
		limit = len(lines)
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("      %s\n", lines[i])
	}

	// Describe the latest backup
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) > 0 {
			latestBackup := fields[0]
			fmt.Printf("\n   🔍 [Diagnosis] Describing latest backup: %s\n", latestBackup)

			descCmd := exec.Command("velero", "describe", "backup", latestBackup)
			descOutput, err := descCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("      ❌ Failed to describe backup: %v\n", err)
				return
			}

			descLines := strings.Split(string(descOutput), "\n")
			for _, line := range descLines {
				fmt.Printf("      %s\n", line)
			}
		}
	}
	fmt.Println("")
}
