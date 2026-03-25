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

type RuleFunc func(alert Alert, server string)

var DiagnosticRules = map[string]RuleFunc{
	"VeleroUnsuccessfulBackup": checkVeleroBackup,
	"ArgoCdAppUnhealthy":       checkArgoUnhealthy,
}

// runCommand executes a shell string locally or remotely over SSH as root
func runCommand(server, cmdStr string) ([]byte, error) {
	if server != "" {
		fmt.Println("      [SSH] (Touch YubiKey if it blinks...)")
		// Use sudo -i to ensure root's PATH and kubeconfig are fully loaded
		remoteCmd := fmt.Sprintf("sudo -i %s", cmdStr)
		// -t forces PTY so PAM can request the YubiKey
		return exec.Command("ssh", "-t", server, remoteCmd).CombinedOutput()
	}
	return exec.Command("sh", "-c", cmdStr).CombinedOutput()
}

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

func checkArgoUnhealthy(alert Alert, server string) {
	fmt.Println("\n   🔍 [Diagnosis] Querying Prometheus for unhealthy ArgoCD apps...")

	promQuery := `argocd_app_info{health_status!="Healthy"}`
	apiPath := fmt.Sprintf("/api/v1/namespaces/monitoring/services/prometheus-operated:9090/proxy/api/v1/query?query=%s", url.QueryEscape(promQuery))

	cmdStr := fmt.Sprintf("kubectl get --raw '%s'", apiPath)

	output, err := runCommand(server, cmdStr)
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for _, res := range resp.Data.Result {
		appName := strings.TrimSpace(res.Metric["name"])
		fmt.Fprintf(w, "      ⚠️  App: %s\t| Health: %s\t| Sync: %s\t| Ns: %s\n",
			appName, res.Metric["health_status"], res.Metric["sync_status"], res.Metric["dest_namespace"])
	}
	w.Flush()
	fmt.Println("")
}

func checkVeleroBackup(alert Alert, server string) {
	fmt.Println("\n   🔍 [Diagnosis] Running: velero get backup (showing top 5)")

	output, err := runCommand(server, "velero get backup")
	if err != nil {
		fmt.Printf("      ❌ Failed to run velero: %v\n", err)
		return
	}

	cleanOutput := strings.ReplaceAll(string(output), "\r", "")
	lines := strings.Split(strings.TrimSpace(cleanOutput), "\n")

	limit := 5
	if len(lines) < limit {
		limit = len(lines)
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("      %s\n", lines[i])
	}

	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) > 0 {
			latestBackup := fields[0]
			fmt.Printf("\n   🔍 [Diagnosis] Describing latest backup: %s\n", latestBackup)

			descCmdStr := fmt.Sprintf("velero describe backup %s --details", latestBackup)
			descOutput, err := runCommand(server, descCmdStr)
			if err != nil {
				fmt.Printf("      ❌ Failed to describe backup: %v\n", err)
				return
			}

			cleanDescOutput := strings.ReplaceAll(string(descOutput), "\r", "")
			descLines := strings.Split(cleanDescOutput, "\n")
			for _, line := range descLines {
				fmt.Printf("      %s\n", line)
			}
		}
	}
	fmt.Println("")
}
