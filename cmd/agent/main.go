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
	// AgentRole determines the agent's behavior:
	// "server" - RDP session host: creates virtual printer, captures spool, uploads to cloud
	// "client" - User workstation: polls for jobs, prints to local printers
	// "standard" - Legacy/both: discovers printers and executes jobs (default)
	AgentRole        string `json:"agent_role"`
	// VirtualPrinterName is the name of the virtual printer created in server mode.
	VirtualPrinterName string `json:"virtual_printer_name"`
	// PrintListenPort is the TCP port the agent listens on for captured print data (server mode).
	PrintListenPort  int    `json:"print_listen_port"`
	// StorageServiceURL is the URL of the OpenPrint storage service for document uploads.
	StorageServiceURL string `json:"storage_service_url"`
	// EnableReceiptPrinter creates an additional receipt/thermal virtual printer in server mode.
	EnableReceiptPrinter bool `json:"enable_receipt_printer"`
	// ReceiptPrinterName is the name of the receipt virtual printer.
	ReceiptPrinterName string `json:"receipt_printer_name"`
	// ReceiptListenPort is the TCP port for the receipt virtual printer.
	ReceiptListenPort int `json:"receipt_listen_port"`
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
	// Server mode fields
	printListener        net.Listener
	receiptPrintListener net.Listener
}

// CapturedPrintJob represents a print job captured from the virtual printer on the RDP session host.
type CapturedPrintJob struct {
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	PrinterType string `json:"printer_type"` // "standard" or "receipt"
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	Title       string `json:"title"`
	PrinterName string `json:"printer_name"`
	SessionID   int    `json:"session_id"`
	Size        int64  `json:"size"`
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
	JobID            string            `json:"job_id"`
	DocumentID       string            `json:"document_id"`
	DocumentURL      string            `json:"document_url"`
	DocumentChecksum string            `json:"document_checksum"`
	PrinterID        string            `json:"printer_id"`
	PrinterName      string            `json:"printer_name"`
	Title            string            `json:"title"`
	Copies           int               `json:"copies"`
	ColorMode        string            `json:"color_mode"`
	Duplex           bool              `json:"duplex"`
	MediaType        string            `json:"media_type"`
	Quality          string            `json:"quality"`
	Options          map[string]string `json:"options,omitempty"`
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
	log.Printf("Starting OpenPrint Agent v%s (role: %s)", a.version, a.getRole())
	log.Printf("Agent ID: %s", a.config.AgentID)
	log.Printf("Server: %s", a.serverURL)

	// Initial printer discovery
	a.discoverPrinters()
	a.registerPrinters()

	// Server mode: set up virtual printer(s) and TCP listener(s) for print capture
	if a.getRole() == "server" {
		// Standard virtual printer (A4/PostScript for documents)
		if err := a.setupVirtualPrinter(ctx); err != nil {
			log.Printf("WARNING: Failed to set up virtual printer: %v", err)
		} else {
			go a.startPrintCaptureListener(ctx, "standard")
		}

		// Receipt virtual printer (narrow paper for thermal/POS printers)
		if a.config.EnableReceiptPrinter {
			if err := a.setupReceiptVirtualPrinter(ctx); err != nil {
				log.Printf("WARNING: Failed to set up receipt virtual printer: %v", err)
			} else {
				go a.startPrintCaptureListener(ctx, "receipt")
			}
		}
	}

	// Start heartbeat goroutine
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Start job polling goroutine (client and standard modes)
	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	// Start printer discovery refresh goroutine
	discoveryTicker := time.NewTicker(5 * time.Minute)
	defer discoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopping...")
			if a.printListener != nil {
				a.printListener.Close()
			}
			if a.receiptPrintListener != nil {
				a.receiptPrintListener.Close()
			}
			if a.getRole() == "server" {
				a.removeVirtualPrinter()
				if a.config.EnableReceiptPrinter {
					a.removeReceiptVirtualPrinter()
				}
			}
			a.stopCh <- struct{}{}
			return
		case <-heartbeatTicker.C:
			if err := a.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		case <-pollTicker.C:
			// Only poll for jobs in client or standard mode
			if a.getRole() != "server" {
				jobs, err := a.pollForJobs()
				if err != nil {
					log.Printf("Job poll failed: %v", err)
				} else {
					a.processJobs(ctx, jobs)
				}
			}
		case <-discoveryTicker.C:
			a.discoverPrinters()
			a.registerPrinters()
		}
	}
}

// getRole returns the agent's role, defaulting to "standard".
func (a *Agent) getRole() string {
	if a.config.AgentRole == "" {
		return "standard"
	}
	return a.config.AgentRole
}

// getVirtualPrinterName returns the virtual printer name for server mode.
func (a *Agent) getVirtualPrinterName() string {
	if a.config.VirtualPrinterName != "" {
		return a.config.VirtualPrinterName
	}
	return "OpenPrint"
}

// getPrintListenPort returns the TCP port for standard print capture.
func (a *Agent) getPrintListenPort() int {
	if a.config.PrintListenPort > 0 {
		return a.config.PrintListenPort
	}
	return 9100
}

// getReceiptPrinterName returns the receipt virtual printer name.
func (a *Agent) getReceiptPrinterName() string {
	if a.config.ReceiptPrinterName != "" {
		return a.config.ReceiptPrinterName
	}
	return "OpenPrint Receipt"
}

// getReceiptListenPort returns the TCP port for receipt print capture.
func (a *Agent) getReceiptListenPort() int {
	if a.config.ReceiptListenPort > 0 {
		return a.config.ReceiptListenPort
	}
	return 9101
}

// virtualPrinterDrivers is the preference order for virtual printer drivers.
// PostScript preserves full formatting, fonts, graphics, and colors — matching
// the fidelity of native Windows RDP local printer redirection (EMF/RAW).
// Falls back to less capable drivers if the preferred one is unavailable.
var virtualPrinterDrivers = []struct {
	Name        string
	ContentType string
}{
	{"Microsoft PS Class Driver", "application/postscript"},
	{"MS Publisher Color Printer", "application/postscript"},
	{"Generic / Text Only", "text/plain"},
}

// setupVirtualPrinter creates a virtual printer that sends print data to a local TCP port.
// It uses a PostScript driver (preferred) to capture full-fidelity output including
// fonts, graphics, layout, and colors — producing output equivalent to what Windows
// RDP local printer redirection provides.
func (a *Agent) setupVirtualPrinter(ctx context.Context) error {
	printerName := a.getVirtualPrinterName()
	port := a.getPrintListenPort()
	portName := fmt.Sprintf("OPENPRINT_127.0.0.1_%d", port)

	log.Printf("Setting up virtual printer '%s' on port %d", printerName, port)

	// Check if printer already exists
	checkCmd := powershellCommand(fmt.Sprintf(`Get-Printer -Name "%s" -ErrorAction SilentlyContinue | Select-Object Name | ConvertTo-Json`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", checkCmd)
	if output, err := cmd.Output(); err == nil && len(output) > 0 && strings.Contains(string(output), printerName) {
		log.Printf("Virtual printer '%s' already exists", printerName)
		return nil
	}

	// Step 1: Create a Standard TCP/IP port pointing to localhost
	addPortCmd := powershellCommand(fmt.Sprintf(
		`Add-PrinterPort -Name "%s" -PrinterHostAddress "127.0.0.1" -PortNumber %d -SNMP $false -ErrorAction Stop`,
		portName, port,
	))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addPortCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Port might already exist, continue
		log.Printf("Add printer port output: %s (err: %v)", string(output), err)
	}

	// Step 2: Find the best available printer driver (prefer PostScript for full fidelity)
	driverName := ""
	for _, driver := range virtualPrinterDrivers {
		checkDriverCmd := powershellCommand(fmt.Sprintf(`Get-PrinterDriver -Name "%s" -ErrorAction SilentlyContinue`, driver.Name))
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", checkDriverCmd)
		if _, err := cmd.Output(); err == nil {
			driverName = driver.Name
			log.Printf("Using printer driver: %s (content type: %s)", driver.Name, driver.ContentType)
			break
		}
		// Try adding the driver
		addDriverCmd := powershellCommand(fmt.Sprintf(`Add-PrinterDriver -Name "%s" -ErrorAction SilentlyContinue`, driver.Name))
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addDriverCmd)
		if err := cmd.Run(); err == nil {
			driverName = driver.Name
			log.Printf("Installed and using printer driver: %s (content type: %s)", driver.Name, driver.ContentType)
			break
		}
	}

	if driverName == "" {
		return fmt.Errorf("no suitable printer driver found; tried: %v", virtualPrinterDrivers)
	}

	// Step 3: Create the virtual printer
	addPrinterCmd := powershellCommand(fmt.Sprintf(
		`Add-Printer -Name "%s" -DriverName "%s" -PortName "%s" -Comment "OpenPrint Cloud Virtual Printer - prints route to your local printer" -ErrorAction Stop`,
		printerName, driverName, portName,
	))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addPrinterCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create virtual printer: %s: %w", string(output), err)
	}

	log.Printf("Virtual printer '%s' created successfully with driver '%s'", printerName, driverName)
	return nil
}

// setupReceiptVirtualPrinter creates a receipt/thermal virtual printer.
// Uses "Generic / Text Only" driver which is ideal for POS/receipt printers:
// applications that print receipts already format their output as plain text
// with the correct narrow width.
func (a *Agent) setupReceiptVirtualPrinter(ctx context.Context) error {
	printerName := a.getReceiptPrinterName()
	port := a.getReceiptListenPort()
	portName := fmt.Sprintf("OPENPRINT_RECEIPT_127.0.0.1_%d", port)

	log.Printf("Setting up receipt virtual printer '%s' on port %d", printerName, port)

	// Check if printer already exists
	checkCmd := powershellCommand(fmt.Sprintf(`Get-Printer -Name "%s" -ErrorAction SilentlyContinue | Select-Object Name | ConvertTo-Json`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", checkCmd)
	if output, err := cmd.Output(); err == nil && len(output) > 0 && strings.Contains(string(output), printerName) {
		log.Printf("Receipt virtual printer '%s' already exists", printerName)
		return nil
	}

	// Create TCP port
	addPortCmd := powershellCommand(fmt.Sprintf(
		`Add-PrinterPort -Name "%s" -PrinterHostAddress "127.0.0.1" -PortNumber %d -SNMP $false -ErrorAction Stop`,
		portName, port,
	))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addPortCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Add receipt printer port output: %s (err: %v)", string(output), err)
	}

	// Receipt printers: use "Generic / Text Only" driver
	// POS/receipt applications format their own output as plain text with proper width
	driverName := "Generic / Text Only"
	checkDriverCmd := powershellCommand(fmt.Sprintf(`Get-PrinterDriver -Name "%s" -ErrorAction SilentlyContinue`, driverName))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", checkDriverCmd)
	if _, err := cmd.Output(); err != nil {
		addDriverCmd := powershellCommand(fmt.Sprintf(`Add-PrinterDriver -Name "%s" -ErrorAction SilentlyContinue`, driverName))
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addDriverCmd)
		cmd.Run()
	}

	// Create the receipt virtual printer
	addPrinterCmd := powershellCommand(fmt.Sprintf(
		`Add-Printer -Name "%s" -DriverName "%s" -PortName "%s" -Comment "OpenPrint Receipt Printer - routes to your local thermal/POS printer" -ErrorAction Stop`,
		printerName, driverName, portName,
	))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", addPrinterCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create receipt virtual printer: %s: %w", string(output), err)
	}

	log.Printf("Receipt virtual printer '%s' created successfully", printerName)
	return nil
}

// removeVirtualPrinter removes the standard virtual printer.
func (a *Agent) removeVirtualPrinter() {
	printerName := a.getVirtualPrinterName()
	port := a.getPrintListenPort()
	portName := fmt.Sprintf("OPENPRINT_127.0.0.1_%d", port)

	log.Printf("Removing virtual printer '%s'", printerName)

	removeCmd := powershellCommand(fmt.Sprintf(`Remove-Printer -Name "%s" -ErrorAction SilentlyContinue`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", removeCmd)
	cmd.Run()

	removePortCmd := powershellCommand(fmt.Sprintf(`Remove-PrinterPort -Name "%s" -ErrorAction SilentlyContinue`, portName))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", removePortCmd)
	cmd.Run()
}

// removeReceiptVirtualPrinter removes the receipt virtual printer.
func (a *Agent) removeReceiptVirtualPrinter() {
	printerName := a.getReceiptPrinterName()
	port := a.getReceiptListenPort()
	portName := fmt.Sprintf("OPENPRINT_RECEIPT_127.0.0.1_%d", port)

	log.Printf("Removing receipt virtual printer '%s'", printerName)

	removeCmd := powershellCommand(fmt.Sprintf(`Remove-Printer -Name "%s" -ErrorAction SilentlyContinue`, printerName))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", removeCmd)
	cmd.Run()

	removePortCmd := powershellCommand(fmt.Sprintf(`Remove-PrinterPort -Name "%s" -ErrorAction SilentlyContinue`, portName))
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", removePortCmd)
	cmd.Run()
}

// startPrintCaptureListener starts a TCP listener for the given printer type.
// printerType is "standard" or "receipt".
func (a *Agent) startPrintCaptureListener(ctx context.Context, printerType string) {
	var port int
	if printerType == "receipt" {
		port = a.getReceiptListenPort()
	} else {
		port = a.getPrintListenPort()
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("ERROR: Failed to start %s print capture listener on %s: %v", printerType, addr, err)
		return
	}

	if printerType == "receipt" {
		a.receiptPrintListener = listener
	} else {
		a.printListener = listener
	}

	log.Printf("%s print capture listener started on %s", printerType, addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("Accept error (%s): %v", printerType, err)
				continue
			}
		}
		go a.handlePrintCapture(ctx, conn, printerType)
	}
}

// handlePrintCapture handles an incoming print data connection from a virtual printer.
// printerType is "standard" or "receipt".
func (a *Agent) handlePrintCapture(ctx context.Context, conn net.Conn, printerType string) {
	defer conn.Close()

	// Set a read deadline to avoid hanging connections
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	// Create temp file to store captured print data
	tempDir := os.TempDir()
	jobID := uuid.New().String()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("openprint_capture_%s_%s.prn", printerType, jobID))

	f, err := os.Create(tempFile)
	if err != nil {
		log.Printf("Failed to create temp file for capture: %v", err)
		return
	}

	// Read all print data from the connection
	bytesWritten, err := io.Copy(f, conn)
	f.Close()
	if err != nil {
		log.Printf("Error reading print data: %v", err)
		os.Remove(tempFile)
		return
	}

	if bytesWritten == 0 {
		os.Remove(tempFile)
		return
	}

	log.Printf("Captured %s print job: %d bytes -> %s", printerType, bytesWritten, tempFile)

	// Detect the content type from the captured data
	contentType := detectPrintDataFormat(tempFile)
	log.Printf("Detected print data format: %s", contentType)

	// Identify the RDP session user who printed
	capturedJob := a.identifyPrintJobOwner(tempFile, bytesWritten)
	capturedJob.FilePath = tempFile
	capturedJob.ContentType = contentType
	capturedJob.PrinterType = printerType

	// Upload to OpenPrint Cloud and create a routed job
	if err := a.uploadCapturedJob(ctx, capturedJob); err != nil {
		log.Printf("Failed to upload captured job: %v", err)
	}

	// Clean up temp file after upload
	os.Remove(tempFile)
}

// detectPrintDataFormat examines the file header to determine the print data format.
func detectPrintDataFormat(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	header := make([]byte, 64)
	n, err := f.Read(header)
	if err != nil || n == 0 {
		return "application/octet-stream"
	}
	header = header[:n]

	headerStr := string(header)

	// PostScript: starts with %!PS or %!
	if strings.HasPrefix(headerStr, "%!PS") || strings.HasPrefix(headerStr, "%!") {
		return "application/postscript"
	}

	// PDF: starts with %PDF
	if strings.HasPrefix(headerStr, "%PDF") {
		return "application/pdf"
	}

	// PCL: starts with ESC character sequences
	if len(header) > 0 && header[0] == 0x1B {
		return "application/vnd.hp-pcl"
	}

	// XPS: ZIP-based format (PK magic bytes)
	if len(header) >= 2 && header[0] == 0x50 && header[1] == 0x4B {
		return "application/oxps"
	}

	return "application/octet-stream"
}

// identifyPrintJobOwner identifies the user who submitted a print job using the Windows spooler.
func (a *Agent) identifyPrintJobOwner(tempFile string, size int64) *CapturedPrintJob {
	printerName := a.getVirtualPrinterName()

	job := &CapturedPrintJob{
		PrinterName: printerName,
		Size:        size,
		FileName:    filepath.Base(tempFile),
		Title:       "Captured Print Job",
	}

	// Query the Windows Print Spooler for the most recent job on our virtual printer
	// This gives us the username and document title
	psCmd := powershellCommand(fmt.Sprintf(
		`Get-PrintJob -PrinterName "%s" -ErrorAction SilentlyContinue | Sort-Object -Property SubmittedTime -Descending | Select-Object -First 1 UserName, DocumentName, SubmittedTime | ConvertTo-Json`,
		printerName,
	))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		var spoolJob struct {
			UserName     string `json:"UserName"`
			DocumentName string `json:"DocumentName"`
		}
		if json.Unmarshal(output, &spoolJob) == nil {
			job.UserName = spoolJob.UserName
			job.Title = spoolJob.DocumentName
		}
	}

	// If we couldn't get the username from the spooler, try the RDP session
	if job.UserName == "" {
		job.UserName = a.getRDPSessionUser()
	}

	// Try to resolve the Windows username to an email
	if job.UserName != "" {
		job.UserEmail = a.resolveUserEmail(job.UserName)
	}

	log.Printf("Print job owner: user=%s email=%s title=%s", job.UserName, job.UserEmail, job.Title)
	return job
}

// getRDPSessionUser returns the username of the active RDP session user.
func (a *Agent) getRDPSessionUser() string {
	// Use qwinsta to enumerate RDP sessions
	cmd := exec.Command("qwinsta")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse qwinsta output to find active RDP sessions
	// Format: SESSIONNAME  USERNAME  ID  STATE  TYPE  DEVICE
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Active") && strings.Contains(strings.ToLower(line), "rdp") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}

	return ""
}

// resolveUserEmail attempts to resolve a Windows username to an email address.
func (a *Agent) resolveUserEmail(username string) string {
	// Strip domain prefix (DOMAIN\username -> username)
	if idx := strings.LastIndex(username, "\\"); idx >= 0 {
		username = username[idx+1:]
	}

	// Try Active Directory lookup via PowerShell
	psCmd := powershellCommand(fmt.Sprintf(
		`try { (Get-ADUser -Identity "%s" -Properties EmailAddress -ErrorAction Stop).EmailAddress } catch { "" }`,
		username,
	))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err == nil {
		email := strings.TrimSpace(string(output))
		if email != "" && strings.Contains(email, "@") {
			return email
		}
	}

	// Fallback: check OpenPrint server for username -> email mapping
	url := fmt.Sprintf("%s/user-printer-mappings/resolve?username=%s", a.serverURL, username)
	resp, err := a.client.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result struct {
				Email string `json:"user_email"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				return result.Email
			}
		}
	}

	return ""
}

// uploadCapturedJob uploads a captured print job to the OpenPrint Cloud.
func (a *Agent) uploadCapturedJob(ctx context.Context, captured *CapturedPrintJob) error {
	log.Printf("Uploading captured job: user=%s title=%s size=%d", captured.UserName, captured.Title, captured.Size)

	// Step 1: Upload document to storage service
	storageURL := a.config.StorageServiceURL
	if storageURL == "" {
		storageURL = a.serverURL
	}

	documentID, checksum, err := a.uploadDocumentToStorage(storageURL, captured)
	if err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}

	log.Printf("Document uploaded: id=%s checksum=%s", documentID, checksum)

	// Step 2: Create a print job in the job service that will be routed to the user's client agent
	// Use printer type-specific routing: __user_default__ for standard, __user_default_receipt__ for receipt
	printerID := "__user_default__"
	mediaType := "a4"
	if captured.PrinterType == "receipt" {
		printerID = "__user_default_receipt__"
		mediaType = "receipt"
	}

	jobReq := map[string]interface{}{
		"document_id": documentID,
		"printer_id":  printerID,
		"user_name":   captured.UserName,
		"user_email":  captured.UserEmail,
		"title":       captured.Title,
		"copies":      1,
		"color_mode":  "auto",
		"media_type":  mediaType,
		"quality":     "normal",
		"options": map[string]string{
			"source":            "rdp_capture",
			"server_agent_id":   a.config.AgentID,
			"captured_size":     fmt.Sprintf("%d", captured.Size),
			"content_type":      captured.ContentType,
			"printer_type":      captured.PrinterType,
		},
	}

	body, _ := json.Marshal(jobReq)
	resp, err := a.client.Post(a.serverURL+"/jobs", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("job creation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var jobResp struct {
		JobID string `json:"job_id"`
	}
	json.NewDecoder(resp.Body).Decode(&jobResp)

	log.Printf("Print job created: id=%s for user=%s", jobResp.JobID, captured.UserEmail)
	return nil
}

// uploadDocumentToStorage uploads a captured document to the storage service.
func (a *Agent) uploadDocumentToStorage(storageURL string, captured *CapturedPrintJob) (string, string, error) {
	f, err := os.Open(captured.FilePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	// Read file content
	fileData, err := io.ReadAll(f)
	if err != nil {
		return "", "", err
	}

	// Compute checksum
	hash := sha256.New()
	hash.Write(fileData)
	checksum := hex.EncodeToString(hash.Sum(nil))

	// Use the detected content type for proper handling on the client side
	contentType := captured.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build the request body for the upload
	uploadReq := map[string]interface{}{
		"name":         captured.Title,
		"content_type": contentType,
		"size":         captured.Size,
		"checksum":     checksum,
		"user_email":   captured.UserEmail,
		"data":         fileData,
	}

	reqBody, _ := json.Marshal(uploadReq)
	resp, err := a.client.Post(storageURL+"/documents", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var uploadResp struct {
		DocumentID string `json:"document_id"`
		Checksum   string `json:"checksum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", "", err
	}

	return uploadResp.DocumentID, uploadResp.Checksum, nil
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
		"agent_role":       a.getRole(),
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

	// Download document with appropriate extension based on content type
	tempDir := os.TempDir()
	ext := ".prn"
	if ct := getJobOption(job, "content_type"); ct != "" {
		switch ct {
		case "application/postscript":
			ext = ".ps"
		case "application/pdf":
			ext = ".pdf"
		case "application/vnd.hp-pcl":
			ext = ".pcl"
		case "application/oxps":
			ext = ".xps"
		}
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("openprint_%s%s", job.JobID, ext))

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
// Handles PostScript, PDF, raw data, and receipt/thermal printer output.
func (a *Agent) printDocument(filePath, printerName string, job PrintJob) error {
	// Check if this is a receipt printer job
	printerType := getJobOption(job, "printer_type")
	if printerType == "receipt" {
		log.Printf("Printing receipt: printer=%s file=%s", printerName, filePath)
		return a.printReceipt(filePath, printerName)
	}

	// Determine the content type from job options or file extension
	contentType := ""
	if job.ColorMode == "auto" {
		// Check job options for content_type hint (set by server agent capture)
		contentType = getJobOption(job, "content_type")
	}
	if contentType == "" {
		contentType = detectPrintDataFormat(filePath)
	}

	log.Printf("Printing document: printer=%s content_type=%s file=%s", printerName, contentType, filePath)

	switch contentType {
	case "application/postscript":
		return a.printPostScript(filePath, printerName, job)
	case "application/pdf":
		return a.printPDF(filePath, printerName, job)
	case "application/vnd.hp-pcl":
		// PCL data can be sent directly to the printer via RAW port
		return a.printRaw(filePath, printerName)
	default:
		// For raw/unknown data, try sending directly to printer
		return a.printRaw(filePath, printerName)
	}
}

// printPostScript prints a PostScript file by converting to PDF first, then printing.
// This ensures the full formatting, fonts, graphics, and colors are preserved.
func (a *Agent) printPostScript(filePath, printerName string, job PrintJob) error {
	// Try Ghostscript to convert PS to PDF, then print the PDF
	gsPath := findGhostscript()
	if gsPath != "" {
		pdfPath := filePath + ".pdf"
		defer os.Remove(pdfPath)

		cmd := exec.Command(gsPath,
			"-dBATCH", "-dNOPAUSE", "-dQUIET",
			"-sDEVICE=pdfwrite",
			"-sOutputFile="+pdfPath,
			filePath,
		)
		if err := cmd.Run(); err == nil {
			return a.printPDF(pdfPath, printerName, job)
		}
		log.Printf("Ghostscript PS->PDF conversion failed, falling back to raw print")
	}

	// Fallback: send PostScript directly to printer (works for PS-capable printers)
	return a.printRaw(filePath, printerName)
}

// printPDF prints a PDF file using the best available method on Windows.
func (a *Agent) printPDF(filePath, printerName string, job PrintJob) error {
	// Method 1: Use SumatraPDF (lightweight, supports silent printing)
	sumatraPath := findSumatraPDF()
	if sumatraPath != "" {
		cmd := exec.Command(sumatraPath,
			"-print-to", printerName,
			"-silent",
			"-print-settings", fmt.Sprintf("copies=%d", max(job.Copies, 1)),
			filePath,
		)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Method 2: Use PowerShell with .NET to print PDF
	psCmd := powershellCommand(fmt.Sprintf(
		`$printJob = Start-Process -FilePath "%s" -Verb PrintTo -ArgumentList "%s" -PassThru -WindowStyle Hidden; `+
			`$printJob | Wait-Process -Timeout 120 -ErrorAction SilentlyContinue`,
		filePath, printerName,
	))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	return cmd.Run()
}

// printRaw sends raw print data directly to the printer's port using PowerShell.
func (a *Agent) printRaw(filePath, printerName string) error {
	// Use PowerShell to send raw data to the printer via Windows spooler
	psCmd := powershellCommand(fmt.Sprintf(
		`$bytes = [System.IO.File]::ReadAllBytes("%s"); `+
			`$printer = Get-Printer -Name "%s"; `+
			`$port = $printer.PortName; `+
			`Copy-Item -Path "%s" -Destination ("\\.\\" + $port) -Force`,
		filePath, printerName, filePath,
	))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	if err := cmd.Run(); err != nil {
		// Fallback: use the Windows print command via file association
		return exec.Command("cmd", "/C",
			fmt.Sprintf(`rundll32 mshtml.dll,PrintHTML "%s"`, filePath),
		).Run()
	}
	return nil
}

// printReceipt sends captured receipt data directly to a thermal/POS printer.
// Receipt data from "Generic / Text Only" driver is plain text that thermal printers
// handle natively. We send it as a raw print job through the Windows spooler.
func (a *Agent) printReceipt(filePath, printerName string) error {
	// Send raw data to the receipt printer via Windows spooler RAW datatype.
	// This preserves the original formatting from the POS/ERP application.
	psCmd := powershellCommand(fmt.Sprintf(
		`$printerPath = (Get-Printer -Name "%s").PortName; `+
			`$rawData = [System.IO.File]::ReadAllBytes("%s"); `+
			`$stream = [System.IO.File]::OpenWrite("\\.\\" + $printerPath); `+
			`$stream.Write($rawData, 0, $rawData.Length); `+
			`$stream.Close()`,
		printerName, filePath,
	))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	if err := cmd.Run(); err != nil {
		// Fallback: use lpr command for network receipt printers
		return exec.Command("lpr",
			"-S", "localhost",
			"-P", printerName,
			"-o", "l", // raw mode
			filePath,
		).Run()
	}
	return nil
}

// findGhostscript looks for Ghostscript executable on the system.
func findGhostscript() string {
	// Check common Ghostscript locations
	paths := []string{
		`C:\Program Files\gs\gs10.0\bin\gswin64c.exe`,
		`C:\Program Files\gs\gs9.56\bin\gswin64c.exe`,
		`C:\Program Files (x86)\gs\gs10.0\bin\gswin32c.exe`,
		`C:\Program Files (x86)\gs\gs9.56\bin\gswin32c.exe`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Try PATH
	if p, err := exec.LookPath("gswin64c.exe"); err == nil {
		return p
	}
	if p, err := exec.LookPath("gswin32c.exe"); err == nil {
		return p
	}
	return ""
}

// findSumatraPDF looks for SumatraPDF executable on the system.
func findSumatraPDF() string {
	paths := []string{
		`C:\Program Files\SumatraPDF\SumatraPDF.exe`,
		`C:\Program Files (x86)\SumatraPDF\SumatraPDF.exe`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("SumatraPDF.exe"); err == nil {
		return p
	}
	return ""
}

// getJobOption extracts an option from the job's options map.
func getJobOption(job PrintJob, key string) string {
	if job.Options == nil {
		return ""
	}
	return job.Options[key]
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
