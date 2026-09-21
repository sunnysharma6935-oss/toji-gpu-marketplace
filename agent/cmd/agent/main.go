package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type GPUInfo struct {
	Index         int
	Model         string
	VRAMMB        int
	DriverVersion string
}

type PendingRental struct {
	RentalID string `json:"rental_id"`
	GPUIndex int    `json:"gpu_index"`
}

// activeRental is this agent PROCESS's in-memory view of rentals it's responsible for
// heartbeating. It is deliberately not persisted — lost on restart, then rebuilt by
// reconcileActiveRentals() comparing local Docker state against the backend's view.
// Flagged rather than hidden: a restart still has a gap between "agent comes back up"
// and "next reconciliation tick" during which a genuinely-running rental gets no
// usage-heartbeats and therefore isn't billed for that window — acceptable for MVP,
// bounded by the 30-second cadence, but a real gap.
type activeRental struct {
	RentalID      string
	ContainerName string
}

var activeRentals = map[string]activeRental{}

func getGPUs() ([]GPUInfo, error) {
	cmd := exec.Command(
		"nvidia-smi",
		"--query-gpu=index,name,memory.total,driver_version",
		"--format=csv,noheader,nounits",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		vram, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			continue
		}
		gpus = append(gpus, GPUInfo{
			Index: index, Model: strings.TrimSpace(parts[1]),
			VRAMMB: vram, DriverVersion: strings.TrimSpace(parts[3]),
		})
	}
	return gpus, nil
}

func sendHeartbeat() {
	// Unchanged: machine liveness only. Never touches a rental or billing.
	token := os.Getenv("AGENT_TOKEN")
	req, err := http.NewRequest("POST", "http://localhost:8080/hosts/heartbeat", nil)
	if err != nil {
		fmt.Println("Heartbeat request failed:", err)
		return
	}
	req.Header.Set("X-Agent-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Heartbeat failed:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Heartbeat status:", resp.Status)
}

// findFreePort asks the OS for an unused TCP port so concurrent rentals never collide
// (the previous version hardcoded port 20000 for every rental — a real bug fixed here,
// not a separate change). Small race window between closing this probe listener and
// Docker binding the same port — accepted for MVP, flagged rather than hidden.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func randomJupyterToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func reportReady(rentalID string, port int, token string) error {
	agentToken := os.Getenv("AGENT_TOKEN")
	payload := map[string]interface{}{"ssh_port": port, "jupyter_token": token}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "http://localhost:8080/agent/rentals/"+rentalID+"/ready", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agentToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ready call failed: %s: %s", resp.Status, string(b))
	}
	return nil
}

func reportProvisioningFailed(rentalID, reason string) {
	agentToken := os.Getenv("AGENT_TOKEN")
	payload := map[string]string{"reason": reason}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "http://localhost:8080/agent/rentals/"+rentalID+"/provisioning-failed", strings.NewReader(string(body)))
	if err != nil {
		fmt.Println("provisioning-failed report build error:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agentToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("provisioning-failed report failed:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Reported provisioning failure for", rentalID, "status:", resp.Status)
}

func stopRentalOnBackend(rentalID, reason string) {
	agentToken := os.Getenv("AGENT_TOKEN")
	payload := map[string]string{"reason": reason}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "http://localhost:8080/agent/rentals/"+rentalID+"/stop", strings.NewReader(string(body)))
	if err != nil {
		fmt.Println("stop report build error:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agentToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("stop report failed:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Reported stop for", rentalID, "status:", resp.Status)
}

func removeContainer(containerName string) {
	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", "-f", containerName).Run()
	fmt.Println("Removed container:", containerName)
}

func startRentalContainer(rental PendingRental) {
	containerName := "gpu-rental-" + rental.RentalID

	checkCmd := exec.Command("docker", "ps", "-a", "--filter", "name=^/"+containerName+"$", "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil {
		fmt.Println("Docker check failed:", err)
		return
	}
	if strings.TrimSpace(string(output)) != "" {
		fmt.Println("Rental container already exists:", containerName)
		return
	}

	port, err := findFreePort()
	if err != nil {
		fmt.Println("Could not find free port:", err)
		reportProvisioningFailed(rental.RentalID, "could not allocate a host port")
		return
	}
	token := randomJupyterToken()

	fmt.Println("Starting Jupyter rental container:", containerName, "GPU:", rental.GPUIndex, "port:", port)

	cmd := exec.Command(
		"docker", "run", "-d",
		"--name", containerName,
		"--gpus", "device="+strconv.Itoa(rental.GPUIndex),
		"-p", fmt.Sprintf("%d:8888", port),
		"quay.io/jupyter/pytorch-notebook:latest",
		"start-notebook.py",
		"--ServerApp.token="+token,
		"--ServerApp.ip=0.0.0.0",
		"--ServerApp.port=8888",
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Docker container start failed:", err)
		fmt.Println("Docker output:", string(output))
		// provisioning -> failed path: attempt cleanup of any partial container first.
		removeContainer(containerName)
		reportProvisioningFailed(rental.RentalID, "docker run failed: "+strings.TrimSpace(string(output)))
		return
	}

	fmt.Println("Jupyter rental container started:", strings.TrimSpace(string(output)))

	if err := reportReady(rental.RentalID, port, token); err != nil {
		fmt.Println("Reporting ready failed, rolling back container:", err)
		removeContainer(containerName)
		reportProvisioningFailed(rental.RentalID, "backend rejected ready report: "+err.Error())
		return
	}

	activeRentals[rental.RentalID] = activeRental{RentalID: rental.RentalID, ContainerName: containerName}
	fmt.Println("Jupyter URL: http://localhost:" + strconv.Itoa(port) + "/lab?token=" + token)
}

func pollPendingRentals() {
	token := os.Getenv("AGENT_TOKEN")
	req, err := http.NewRequest("GET", "http://localhost:8080/agent/rentals/pending", nil)
	if err != nil {
		fmt.Println("Pending-rental request failed:", err)
		return
	}
	req.Header.Set("X-Agent-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Pending-rental poll failed:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Pending-rental body read error:", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Pending-rental raw response:", string(body))
		return
	}

	var pending []PendingRental
	if err := json.Unmarshal(body, &pending); err != nil {
		fmt.Println("Pending-rental JSON error:", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	fmt.Println("Pending rentals found:", pending)
	for _, rental := range pending {
		startRentalContainer(rental)
	}
}

// sendUsageHeartbeats is the billing trigger — one call per rental this agent process
// believes is running, on the SAME 30-second ticker as sendHeartbeat/pollPendingRentals.
// No second timing loop.
func sendUsageHeartbeats() {
	agentToken := os.Getenv("AGENT_TOKEN")
	for rentalID, info := range activeRentals {
		req, err := http.NewRequest("POST", "http://localhost:8080/agent/rentals/"+rentalID+"/usage-heartbeat", nil)
		if err != nil {
			fmt.Println("usage-heartbeat build error:", err)
			continue
		}
		req.Header.Set("X-Agent-Token", agentToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("usage-heartbeat failed for", rentalID, ":", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Println("usage-heartbeat non-OK for", rentalID, ":", resp.Status, string(body))
			continue
		}

		var result struct {
			MustStop     bool  `json:"must_stop"`
			ChargedPaise int64 `json:"charged_paise"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("usage-heartbeat parse error:", err)
			continue
		}

		if result.MustStop {
			// Insufficient balance: stop the container immediately, in this same tick,
			// rather than waiting for the next reconciliation pass.
			fmt.Println("Insufficient balance — stopping rental immediately:", rentalID)
			removeContainer(info.ContainerName)
			stopRentalOnBackend(rentalID, "insufficient_balance")
			delete(activeRentals, rentalID)
		}
	}
}

// reconcileActiveRentals compares local Docker state against the backend's view of
// which rentals SHOULD have a container running on this host. Handles two cases:
//  1. A container exists locally for a rental the backend no longer considers active
//     (customer stopped it via the dashboard) -> tear it down.
//  2. The backend considers a rental active but this agent process has no in-memory
//     record of it (agent restarted) -> re-adopt it so it starts getting billed again.
func reconcileActiveRentals() {
	agentToken := os.Getenv("AGENT_TOKEN")
	req, err := http.NewRequest("GET", "http://localhost:8080/agent/rentals/active", nil)
	if err != nil {
		fmt.Println("reconcile build error:", err)
		return
	}
	req.Header.Set("X-Agent-Token", agentToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("reconcile request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Println("reconcile non-OK:", resp.Status, string(body))
		return
	}

	var result struct {
		ActiveRentalIDs []string `json:"active_rental_ids"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("reconcile parse error:", err)
		return
	}
	backendActive := map[string]bool{}
	for _, id := range result.ActiveRentalIDs {
		backendActive[id] = true
	}

	listCmd := exec.Command("docker", "ps", "-a", "--filter", "name=gpu-rental-", "--format", "{{.Names}}")
	out, err := listCmd.Output()
	if err != nil {
		fmt.Println("docker ps for reconciliation failed:", err)
		return
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name == "" {
			continue
		}
		rentalID := strings.TrimPrefix(name, "gpu-rental-")
		if !backendActive[rentalID] {
			fmt.Println("Reconciliation: removing orphaned container for stopped rental:", rentalID)
			removeContainer(name)
			delete(activeRentals, rentalID)
		} else if _, tracked := activeRentals[rentalID]; !tracked {
			activeRentals[rentalID] = activeRental{RentalID: rentalID, ContainerName: name}
			fmt.Println("Reconciliation: re-adopted rental into heartbeat tracking:", rentalID)
		}
	}
}

func reportGPU(gpu GPUInfo) {
	token := os.Getenv("AGENT_TOKEN")
	payload := map[string]interface{}{
		"gpu_index": gpu.Index, "model": gpu.Model,
		"vram_mb": gpu.VRAMMB, "driver_version": gpu.DriverVersion,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("GPU report JSON error:", err)
		return
	}
	req, err := http.NewRequest("POST", "http://localhost:8080/hosts/gpu-report", strings.NewReader(string(body)))
	if err != nil {
		fmt.Println("GPU report request failed:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("GPU report failed:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("GPU report:", gpu.Index, gpu.Model, gpu.VRAMMB, "MB status:", resp.Status)
}

func reportGPUs() {
	gpus, err := getGPUs()
	if err != nil {
		fmt.Println("GPU detection failed:", err)
		return
	}
	if len(gpus) == 0 {
		fmt.Println("No GPUs detected.")
		return
	}
	for _, gpu := range gpus {
		reportGPU(gpu)
	}
}

func main() {
	fmt.Println("GPU Marketplace Agent starting...")

	reportGPUs()
	sendHeartbeat()
	pollPendingRentals()
	reconcileActiveRentals() // populate activeRentals on startup, not just on restart-gap

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sendHeartbeat()
		pollPendingRentals()
		sendUsageHeartbeats()
		reconcileActiveRentals()
	}
}
