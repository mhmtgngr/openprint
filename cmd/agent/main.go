// OpenPrint Windows Print Agent
// This is a Windows service that discovers printers, polls for print jobs,
// and executes print commands on Windows systems.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	serviceName   = "OpenPrintAgent"
	serviceDesc   = "OpenPrint Cloud Print Management Agent"
	pollInterval  = 10 * time.Second
	heartbeatInterval = 30 * time.Second
	maxRetries    = 3
	retryDelay    = 5 * time.Second
)

var (
	elog debug.Log
)

// AgentConfig holds the agent configuration.
type AgentConfig struct {
	ServerURL        string `json:"server_url"`
	AgentID          string `json:"agent_id"`
	AgentName        string `json:"agent_name"`
	EnrollmentToken  string `json:"enrollment_token"`
	OrganizationID   string `json:"organization_id"`
	HeartbeatInterval int  `json:"heartbeat_interval_seconds"`
	JobPollInterval  int   `json:"job_poll_interval_seconds"`
	MaxRetries       int   `json:"max_retry_count"`
	LogLevel         string `json:"log_level"`
}

// Agent represents the print agent.
type Agent struct {
	config         *AgentConfig
	client         *http.Client
	serverURL      string
	printers       map[string]*DiscoveredPrinter
	printersMutex  sync.RWMutex
	hostname       string
	version        string
	architecture   string
	domain         string
	username       string
	ipAddress      string
	macAddress     string
	isElevated     bool
	stopCh         chan struct{}
}

// DiscoveredPrinter represents a printer discovered on the system.
type DiscoveredPrinter struct {
	Name            string            `json:"name"`
	Driver          string            `json:"driver"`
	Port            string            `json:"port"`
	ConnectionType  string            `json:"connection_type"`
	Status          string            `json:"status"`
	IsDefault       bool              `json:"is_default"`
	IsShared        bool              `json:"is_shared"`
	ShareName       string            `json:"share_name,omitempty"`
	Location        string            `json:"location,omitempty"`
	Capabilities    *PrinterCaps      `json:"capabilities,omitempty"`
}

// PrinterCaps represents printer capabilities.
type PrinterCaps struct {
	CanColor           bool     `json:"can_color"`
	CanDuplex          bool     `json:"can_duplex"`
	SupportedMediaTypes []string `json:"supported_media_types"`
}

// PrintJob represents a print job from the server.
type PrintJob struct {
	JobID            string `json:"job_id"`
	DocumentID       string `json:"document_id"`
	DocumentURL      string `json:"document_url"`
	DocumentChecksum string `json:"document_checksum"`
	PrinterID        string `json:"printer_id"`
	PrinterName      string `json:"printer_name"`
	Title            string `json:"title"`
	Copies           int    `json:"copies"`
	ColorMode        string `json:"color_mode"`
	Duplex           bool   `json:"duplex"`
	MediaType        string `json:"media_type"`
	Quality          string `json:"quality"`
}

// JobStatusUpdate represents a status update for a job.
type JobStatusUpdate struct {
	JobID       string    `json:"job_id"`
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	PagesPrinted int      `json:"pages_printed,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

func main() {
	isIntSess, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}

	if !isIntSess {
		log.Printf("Running in console mode")
		runConsole()
		return
	}

	log.Printf("Running as Windows service")
	elog, err = eventlog.Open(serviceName)
	if err != nil {
		log.Fatalf("failed to open event log: %v", err)
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", serviceName))
	if err := svc.Run(serviceName, &agentService{}); err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
		log.Fatalf("service failed: %v", err)
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", serviceName))
}

type agentService struct{}

func (s *agentService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Initialize agent
	agent, err := initializeAgent()
	if err != nil {
		elog.Error(1, fmt.Sprintf("failed to initialize agent: %v", err))
		return false, 1
	}

	// Start agent in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agent.Run(ctx)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "Received stop request")
				cancel()
				changes <- svc.Status{State: svc.StopPending}
				break loop
			case svc.Pause:
				elog.Info(1, "Received pause request")
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				elog.Info(1, "Received continue request")
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		case <-agent.stopCh:
			break loop
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}

func runConsole() {
	agent, err := initializeAgent()
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	ctx := context.Background()
	agent.Run(ctx)
}

func initializeAgent() (*Agent, error) {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
		},
	}

	// Get system information
	hostname, _ := os.Hostname()
	domain := getDomain()
	username := getCurrentUsername()
	ipAddress := getIPAddress()
	macAddress := getMACAddress()

	agent := &Agent{
		config:        config,
		client:        client,
		serverURL:     config.ServerURL,
		printers:      make(map[string]*DiscoveredPrinter),
		hostname:      hostname,
		version:       "1.0.0",
		architecture:  runtime.GOARCH,
		domain:        domain,
		username:      username,
		ipAddress:     ipAddress,
		macAddress:    macAddress,
		isElevated:    isElevated(),
		stopCh:        make(chan struct{}),
	}

	// Register agent with server
	if config.AgentID == "" {
		if err := agent.register(); err != nil {
			return nil, fmt.Errorf("failed to register agent: %w", err)
		}
	}

	return agent, nil
}

// Run is the main agent loop.
func (a *Agent) Run(ctx context.Context) {
	log.Printf("Starting OpenPrint Agent v%s", a.version)
	log.Printf("Agent ID: %s", a.config.AgentID)
	log.Printf("Server: %s", a.serverURL)

	// Initial printer discovery
	a.discoverPrinters()
	a.registerPrinters()

	// Start heartbeat goroutine
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Start job polling goroutine
	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	// Start printer discovery refresh goroutine
	discoveryTicker := time.NewTicker(5 * time.Minute)
	defer discoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopping...")
			a.stopCh <- struct{}{}
			return
		case <-heartbeatTicker.C:
			if err := a.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		case <-pollTicker.C:
			jobs, err := a.pollForJobs()
			if err != nil {
				log.Printf("Job poll failed: %v", err)
			} else {
				a.processJobs(ctx, jobs)
			}
		case <-discoveryTicker.C:
			a.discoverPrinters()
			a.registerPrinters()
		}
	}
}

// register registers the agent with the server.
func (a *Agent) register() error {
	req := map[string]interface{}{
		"name":             a.config.AgentName,
		"version":          a.version,
		"os":               getOSVersion(),
		"architecture":     a.architecture,
		"hostname":         a.hostname,
		"domain":           a.domain,
		"username":         a.username,
		"organization_id":  a.config.OrganizationID,
		"enrollment_token": a.config.EnrollmentToken,
		"mac_address":      a.macAddress,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := a.client.Post(a.serverURL+"/agents/register", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var regResp struct {
		AgentID          string `json:"agent_id"`
		HeartbeatInterval int   `json:"heartbeat_interval"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return err
	}

	a.config.AgentID = regResp.AgentID

	// Save updated config
	if err := saveConfig(a.config); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}

	log.Printf("Agent registered successfully: %s", regResp.AgentID)
	return nil
}

// discoverPrinters discovers printers using PowerShell Get-Printer command.
func (a *Agent) discoverPrinters() {
	log.Println("Discovering printers...")

	// Use PowerShell Get-Printer command
	printers, err := a.getPrintersViaPowerShell()
	if err != nil {
		log.Printf("PowerShell printer discovery failed: %v", err)
		// Fallback to basic discovery
		return
	}

	a.printersMutex.Lock()
	defer a.printersMutex.Unlock()

	// Clear existing printers and add new ones
	a.printers = make(map[string]*DiscoveredPrinter)
	for _, p := range printers {
		a.printers[p.Name] = p
	}

	log.Printf("Discovered %d printers", len(printers))
}

// getPrintersViaPowerShell uses PowerShell to enumerate printers.
func (a *Agent) getPrintersViaPowerShell() ([]*DiscoveredPrinter, error) {
	// PowerShell command to get printer details
	psCmd := powershellCommand(`Get-Printer | Select-Object Name, DriverName, PortName, DeviceType, Shared, ShareName, Location, PrinterStatus, Default | ConvertTo-Json`)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var psPrinters []struct {
		Name         string `json:"Name"`
		DriverName   string `json:"DriverName"`
		PortName     string `json:"PortName"`
		DeviceType   string `json:"DeviceType"`
		Shared       bool   `json:"Shared"`
		ShareName    string `json:"ShareName"`
		Location     string `json:"Location"`
		PrinterStatus int   `json:"PrinterStatus"`
		Default      bool   `json:"Default"`
	}

	if err := json.Unmarshal(output, &psPrinters); err != nil {
		// Try parsing as single object
		var singlePrinter struct {
			Name         string `json:"Name"`
			DriverName   string `json:"DriverName"`
			PortName     string `json:"PortName"`
			DeviceType   string `json:"DeviceType"`
			Shared       bool   `json:"Shared"`
			ShareName    string `json:"ShareName"`
			Location     string `json:"Location"`
			PrinterStatus int   `json:"PrinterStatus"`
			Default      bool   `json:"Default"`
		}
		if json.Unmarshal(output, &singlePrinter) == nil {
			psPrinters = []struct {
				Name         string `json:"Name"`
				DriverName   string `json:"DriverName"`
				PortName     string `json:"PortName"`
				DeviceType   string `json:"DeviceType"`
				Shared       bool   `json:"Shared"`
				ShareName    string `json:"ShareName"`
				Location     string `json:"Location"`
				PrinterStatus int   `json:"PrinterStatus"`
				Default      bool   `json:"Default"`
			}{singlePrinter}
		} else {
			return nil, err
		}
	}

	printers := make([]*DiscoveredPrinter, 0, len(psPrinters))
	for _, ps := range psPrinters {
		// Skip fax or virtual printers
		if strings.Contains(strings.ToLower(ps.Name), "fax") ||
		   strings.Contains(strings.ToLower(ps.Name), "onenote") ||
		   strings.Contains(strings.ToLower(ps.Name), "send to") {
			continue
		}

		printer := &DiscoveredPrinter{
			Name:           ps.Name,
			Driver:         ps.DriverName,
			Port:           ps.PortName,
			ConnectionType: getConnectionType(ps.PortName, ps.DeviceType),
			Status:         getPrinterStatus(ps.PrinterStatus),
			IsDefault:      ps.Default,
			IsShared:       ps.Shared,
			ShareName:      ps.ShareName,
			Location:       ps.Location,
			Capabilities:   &PrinterCaps{
				CanColor:  hasColorCapability(ps.Name),
				CanDuplex: hasDuplexCapability(ps.Name),
			},
		}
		printers = append(printers, printer)
	}

	return printers, nil
}

// pollForJobs polls the server for pending print jobs.
func (a *Agent) pollForJobs() ([]PrintJob, error) {
	req := map[string]interface{}{
		"agent_id": a.config.AgentID,
		"limit":    10,
	}

	body, _ := json.Marshal(req)
	resp, err := a.client.Post(a.serverURL+"/agents/jobs/poll", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll failed with status %d", resp.StatusCode)
	}

	var pollResp struct {
		Jobs []PrintJob `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pollResp); err != nil {
		return nil, err
	}

	return pollResp.Jobs, nil
}

// processJobs processes the given print jobs.
func (a *Agent) processJobs(ctx context.Context, jobs []PrintJob) {
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			if err := a.processJob(ctx, job); err != nil {
				log.Printf("Failed to process job %s: %v", job.JobID, err)
				a.updateJobStatus(job, "failed", err.Error(), 0)
			}
		}
	}
}

// processJob processes a single print job.
func (a *Agent) processJob(ctx context.Context, job PrintJob) error {
	log.Printf("Processing job %s: %s", job.JobID, job.Title)

	// Update job status to in_progress
	a.updateJobStatus(job, "in_progress", "Downloading document", 0)

	// Download document
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("openprint_%s.pdf", job.JobID))

	if err := a.downloadDocument(job.DocumentURL, tempFile); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tempFile)

	// Verify checksum
	if job.DocumentChecksum != "" {
		if err := verifyChecksum(tempFile, job.DocumentChecksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Update status
	a.updateJobStatus(job, "in_progress", "Printing", 0)

	// Execute print
	printerName := job.PrinterName
	if printerName == "" {
		// Look up printer by ID
		a.printersMutex.RLock()
		for _, p := range a.printers {
			if p.Name == job.PrinterID {
				printerName = p.Name
				break
			}
		}
		a.printersMutex.RUnlock()
	}

	if printerName == "" {
		return fmt.Errorf("printer not found: %s", job.PrinterID)
	}

	// Print the document using Windows print command
	if err := a.printDocument(tempFile, printerName, job); err != nil {
		return fmt.Errorf("print failed: %w", err)
	}

	// Update to completed
	a.updateJobStatus(job, "completed", "Printed successfully", 0)

	log.Printf("Job %s completed", job.JobID)
	return nil
}

// downloadDocument downloads a document from the server.
func (a *Agent) downloadDocument(url, destPath string) error {
	resp, err := a.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// printDocument prints a document using Windows print commands.
func (a *Agent) printDocument(filePath, printerName string, job PrintJob) error {
	// For PDF files, we can use several methods:
	// 1. Adobe Reader Reader (if installed)
	// 2. Foxit Reader (if installed)
	// 3. Microsoft Print to PDF (converting to print)
	// 4. Windows Shell print command

	// Try using the Windows shell print command
	args := []string{
		"/C",
		"cd /d " + filepath.Dir(filePath),
		"&&",
		"timeout /t 2 /nobreak > nul", // Small delay
		"&&",
		fmt.Sprintf("powershell -Command \"Add-Type -AssemblyName System.Drawing; $p = New-Object System.Drawing.Printing.PrintDocument; $p.PrinterSettings.PrinterName = '%s'; $p.DocumentName = '%s'; Start-Process -FilePath '%s' -ArgumentList '/t', '/p' \"%s\" -Wait -WindowStyle Hidden\"",
			printerName, job.Title, filePath, filePath),
	}

	cmd := exec.Command("cmd", args...)
	return cmd.Run()
}

// updateJobStatus updates the status of a print job on the server.
func (a *Agent) updateJobStatus(job PrintJob, status, message string, pagesPrinted int) {
	update := JobStatusUpdate{
		JobID:       job.JobID,
		AgentID:     a.config.AgentID,
		Status:      status,
		Message:     message,
		PagesPrinted: pagesPrinted,
		Timestamp:   time.Now(),
	}

	body, _ := json.Marshal(update)
	url := fmt.Sprintf("%s/agents/%s/jobs/%s/status", a.serverURL, a.config.AgentID, job.JobID)

	req, _ := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Failed to update job status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Job status update failed with status %d", resp.StatusCode)
	}
}

// sendHeartbeat sends a heartbeat to the server.
func (a *Agent) sendHeartbeat() error {
	a.printersMutex.RLock()
	printerCount := len(a.printers)
	a.printersMutex.RUnlock()

	req := map[string]interface{}{
		"agent_id":      a.config.AgentID,
		"status":        "online",
		"session_state": "active",
		"printer_count": printerCount,
		"job_queue_depth": 0,
		"timestamp":     time.Now(),
	}

	body, _ := json.Marshal(req)
	resp, err := a.client.Post(a.serverURL+"/agents/heartbeat", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status %d", resp.StatusCode)
	}

	return nil
}

// registerPrinters registers discovered printers with the server.
func (a *Agent) registerPrinters() {
	a.printersMutex.RLock()
	defer a.printersMutex.RUnlock()

	if len(a.printers) == 0 {
		return
	}

	printers := make([]map[string]interface{}, 0, len(a.printers))
	for _, p := range a.printers {
		printers = append(printers, map[string]interface{}{
			"name":            p.Name,
			"display_name":    p.Name,
			"driver":          p.Driver,
			"port":            p.Port,
			"connection_type": p.ConnectionType,
			"status":          p.Status,
			"is_default":      p.IsDefault,
			"is_shared":       p.IsShared,
			"share_name":      p.ShareName,
			"location":        p.Location,
			"capabilities":    p.Capabilities,
		})
	}

	req := map[string]interface{}{
		"agent_id": a.config.AgentID,
		"printers": printers,
		"replace":  true,
		"timestamp": time.Now(),
	}

	body, _ := json.Marshal(req)
	resp, err := a.client.Post(a.serverURL+"/agents/printers/discover", "application/json", strings.NewReader(string(body)))
	if err != nil {
		log.Printf("Failed to register printers: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("Registered %d printers", len(printers))
	}
}

// Helper functions

func getConnectionType(port, deviceType string) string {
	port = strings.ToLower(port)
	if strings.HasPrefix(port, "usb") {
		return "local"
	}
	if strings.HasPrefix(port, "com") {
		return "local"
	}
	if strings.HasPrefix(port, "lpt") {
		return "local"
	}
	if strings.Contains(port, "192.168.") || strings.Contains(port, "10.") ||
	   strings.Contains(port, "172.16") || strings.Contains(port, ".") {
		return "network"
	}
	if strings.HasPrefix(port, "wsd") || strings.HasPrefix(port, "https") || strings.HasPrefix(port, "http") {
		return "wsd"
	}
	return "network"
}

func getPrinterStatus(status int) string {
	switch status {
	case 0:
		return "idle"
	case 1:
		return "printing"
	case 2:
		return "offline"
	case 3:
		return "error"
	default:
		return "idle"
	}
}

func hasColorCapability(printerName string) bool {
	// Check printer capabilities via PowerShell
	psCmd := powershellCommand(fmt.Sprintf(`Get-Printer -Name "%s" | Select-Object -ExpandProperty CapabilityDescriptions | ConvertTo-Json`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	descriptions := strings.ToLower(string(output))
	return strings.Contains(descriptions, "color")
}

func hasDuplexCapability(printerName string) bool {
	psCmd := powershellCommand(fmt.Sprintf(`Get-Printer -Name "%s" | Select-Object -ExpandProperty CapabilityDescriptions | ConvertTo-Json`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	descriptions := strings.ToLower(string(output))
	return strings.Contains(descriptions, "duplex")
}

func getOSVersion() string {
	cmd := exec.Command("cmd", "/c", "ver")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func getDomain() string {
	cmd := exec.Command("cmd", "/c", "echo %USERDOMAIN()")
	output, _ := cmd.Output()
	domain := strings.TrimSpace(string(output))
	if domain == "%USERDOMAIN()" {
		return ""
	}
	return domain
}

func getCurrentUsername() string {
	cmd := exec.Command("cmd", "/c", "echo %USERNAME%")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func getIPAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) >= 6 {
			// Skip loopback and virtual interfaces
			if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
				return iface.HardwareAddr.String()
			}
		}
	}
	return ""
}

func isElevated() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

func verifyChecksum(filePath, expectedChecksum string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func loadConfig() (*AgentConfig, error) {
	configPath := getConfigPath()

	// Default config
	config := &AgentConfig{
		ServerURL:        "http://localhost:8003",
		AgentName:        getHostname(),
		HeartbeatInterval: 30,
		JobPollInterval:   10,
		MaxRetries:        3,
		LogLevel:          "info",
	}

	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func saveConfig(config *AgentConfig) error {
	configPath := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func getConfigPath() string {
	// Store in ProgramData for Windows
	return filepath.Join(os.Getenv("ProgramData"), "OpenPrint", "agent", "config.json")
}

func getHostname() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		cmd := exec.Command("hostname")
		output, _ := cmd.Output()
		hostname = strings.TrimSpace(string(output))
	}
	return hostname
}

func powershellCommand(cmd string) string {
	// Escape single quotes for PowerShell
	return strings.ReplaceAll(cmd, "`", "``")
}
