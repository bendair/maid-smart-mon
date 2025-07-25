package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SmartAttribute represents a SMART attribute from smartctl
type SmartAttribute struct {
	ID     int                    `json:"id"`
	Name   string                 `json:"name"`
	Value  int                    `json:"value"`
	Worst  int                    `json:"worst"`
	Thresh int                    `json:"thresh"`
	Raw    map[string]interface{} `json:"raw"`
	Flags  map[string]bool        `json:"flags"`
}

// SmartData represents the JSON output from smartctl
type SmartData struct {
	ATASmartAttributes struct {
		Table []SmartAttribute `json:"table"`
	} `json:"ata_smart_attributes"`
	ModelName    string `json:"model_name"`
	SerialNumber string `json:"serial_number"`
}

// DeviceInfo holds basic device information
type DeviceInfo struct {
	Device       string
	SerialNumber string
	Model        string
	IsMounted    bool
	SmartEnabled bool
}

// HealthAlert represents a health alert
type HealthAlert struct {
	Device        string
	AttributeName string
	AlertType     string
	Message       string
	Timestamp     time.Time
}

// MAIDSmartMonitor is the main monitoring system
type MAIDSmartMonitor struct {
	db            *sql.DB
	dbPath        string
	targetAttribs map[int]string
	logger        *log.Logger
}

// NewMAIDSmartMonitor creates a new monitor instance
func NewMAIDSmartMonitor(dbPath string) (*MAIDSmartMonitor, error) {
	// Target SMART attributes for monitoring
	targetAttribs := map[int]string{
		1:   "Raw_Read_Error_Rate",
		3:   "Spin_Up_Time",
		4:   "Start_Stop_Count",
		5:   "Reallocated_Sector_Ct",
		7:   "Seek_Error_Rate",
		9:   "Power_On_Hours",
		12:  "Power_Cycle_Count",
		187: "Reported_Uncorrectable_Errors",
		188: "Command_Timeout",
		190: "Airflow_Temperature_Cel",
		191: "G_Sense_Error_Rate",
		192: "Power_Off_Retract_Count",
		193: "Load_Cycle_Count",
		194: "Temperature_Celsius",
		196: "Reallocation_Event_Count",
		197: "Current_Pending_Sector",
		198: "Offline_Uncorrectable",
		199: "UDMA_CRC_Error_Count",
		222: "Loaded_Hours",
		240: "Head_Flying_Hours",
		241: "Total_LBAs_Written",
		242: "Total_LBAs_Read",
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	monitor := &MAIDSmartMonitor{
		db:            db,
		dbPath:        dbPath,
		targetAttribs: targetAttribs,
		logger:        log.New(os.Stdout, "[MAID-SMART] ", log.LstdFlags),
	}

	if err := monitor.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return monitor, nil
}

// Close closes the database connection
func (m *MAIDSmartMonitor) Close() error {
	return m.db.Close()
}

// initDatabase initializes the SQLite database with required tables
func (m *MAIDSmartMonitor) initDatabase() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS smart_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device TEXT NOT NULL,
			serial_number TEXT,
			model TEXT,
			timestamp DATETIME NOT NULL,
			attribute_id INTEGER NOT NULL,
			attribute_name TEXT NOT NULL,
			raw_value INTEGER,
			normalized_value INTEGER,
			threshold INTEGER,
			worst_value INTEGER,
			flags TEXT,
			UNIQUE(device, timestamp, attribute_id)
		)`,
		`CREATE TABLE IF NOT EXISTS device_status (
			device TEXT PRIMARY KEY,
			serial_number TEXT,
			model TEXT,
			last_seen DATETIME,
			is_mounted BOOLEAN,
			mount_point TEXT,
			smart_enabled BOOLEAN,
			last_smart_check DATETIME,
			spin_up_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS health_alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device TEXT NOT NULL,
			attribute_name TEXT NOT NULL,
			alert_type TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			resolved BOOLEAN DEFAULT FALSE
		)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	m.logger.Printf("Database initialized: %s", m.dbPath)
	return nil
}

// getMountedDrives returns list of currently mounted drives to avoid spinning up idle disks
func (m *MAIDSmartMonitor) getMountedDrives() ([]string, error) {
	content, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/mounts: %v", err)
	}

	var mountedDrives []string
	deviceMap := make(map[string]bool)

	// Regex to match device names like /dev/sda1, /dev/nvme0n1p1, etc.
	deviceRegex := regexp.MustCompile(`^(/dev/[a-z]+)`)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := fields[0]
			if strings.HasPrefix(device, "/dev/sd") || strings.HasPrefix(device, "/dev/nvme") {
				// Extract base device name (e.g., /dev/sda1 -> /dev/sda)
				matches := deviceRegex.FindStringSubmatch(device)
				if len(matches) > 1 {
					baseDevice := matches[1]
					// Remove partition numbers for SATA drives
					baseDeviceClean := regexp.MustCompile(`\d+$`).ReplaceAllString(baseDevice, "")
					if !deviceMap[baseDeviceClean] {
						deviceMap[baseDeviceClean] = true
						mountedDrives = append(mountedDrives, baseDeviceClean)
					}
				}
			}
		}
	}

	m.logger.Printf("Found %d mounted drives: %v", len(mountedDrives), mountedDrives)
	return mountedDrives, nil
}

// checkSmartSupport checks if device supports SMART without spinning it up
func (m *MAIDSmartMonitor) checkSmartSupport(device string) bool {
	cmd := exec.Command("smartctl", "--nocheck=standby", "-i", device)
	output, err := cmd.Output()
	if err != nil {
		m.logger.Printf("SMART support check failed for %s: %v", device, err)
		return false
	}

	return strings.Contains(string(output), "SMART support is: Enabled")
}

// getDeviceInfo gets device serial number and model without spinning up
func (m *MAIDSmartMonitor) getDeviceInfo(device string) (string, string, error) {
	cmd := exec.Command("smartctl", "--nocheck=standby", "-i", device)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get device info: %v", err)
	}

	var serial, model string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Serial Number:") {
			parts := strings.Split(line, "Serial Number:")
			if len(parts) > 1 {
				serial = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Device Model:") || strings.Contains(line, "Model Number:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				model = strings.TrimSpace(parts[1])
			}
		}
	}

	return serial, model, nil
}

// isDeviceInStandby checks if device is in standby mode
func (m *MAIDSmartMonitor) isDeviceInStandby(device string) bool {
	cmd := exec.Command("smartctl", "--nocheck=standby", "-n", "standby", device)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "STANDBY")
}

// collectSmartData collects SMART data from a device (only if already spinning)
func (m *MAIDSmartMonitor) collectSmartData(device string) (*SmartData, error) {
	// First check if device is in standby mode
	if m.isDeviceInStandby(device) {
		m.logger.Printf("Device %s is in standby mode - skipping to avoid spin-up", device)
		return nil, nil
	}

	// Device is already spinning, safe to collect SMART data
	cmd := exec.Command("smartctl", "-A", "--json", device)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to collect SMART data: %v", err)
	}

	var smartData SmartData
	if err := json.Unmarshal(output, &smartData); err != nil {
		return nil, fmt.Errorf("failed to parse SMART JSON: %v", err)
	}

	return &smartData, nil
}

// parseSmartAttributes parses and filters SMART attributes for target IDs
func (m *MAIDSmartMonitor) parseSmartAttributes(smartData *SmartData, device string) []map[string]interface{} {
	var attributes []map[string]interface{}

	for _, attr := range smartData.ATASmartAttributes.Table {
		if name, exists := m.targetAttribs[attr.ID]; exists {
			rawValue := int64(0)
			if val, ok := attr.Raw["value"]; ok {
				switch v := val.(type) {
				case float64:
					rawValue = int64(v)
				case int64:
					rawValue = v
				case string:
					if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
						rawValue = parsed
					}
				}
			}

			attributes = append(attributes, map[string]interface{}{
				"device":           device,
				"attribute_id":     attr.ID,
				"attribute_name":   name,
				"raw_value":        rawValue,
				"normalized_value": attr.Value,
				"threshold":        attr.Thresh,
				"worst_value":      attr.Worst,
				"flags":            fmt.Sprintf("%+v", attr.Flags),
			})
		}
	}

	return attributes
}

// storeSmartData stores SMART attributes in database
func (m *MAIDSmartMonitor) storeSmartData(attributes []map[string]interface{}, serial, model string) error {
	if len(attributes) == 0 {
		return nil
	}

	timestamp := time.Now()

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO smart_data 
		(device, serial_number, model, timestamp, attribute_id, attribute_name,
		 raw_value, normalized_value, threshold, worst_value, flags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for _, attr := range attributes {
		_, err := stmt.Exec(
			attr["device"], serial, model, timestamp,
			attr["attribute_id"], attr["attribute_name"],
			attr["raw_value"], attr["normalized_value"],
			attr["threshold"], attr["worst_value"], attr["flags"],
		)
		if err != nil {
			return fmt.Errorf("failed to insert attribute: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	m.logger.Printf("Stored %d SMART attributes for %s", len(attributes), attributes[0]["device"])
	return nil
}

// updateDeviceStatus updates device status in database
func (m *MAIDSmartMonitor) updateDeviceStatus(device, serial, model string, isMounted, smartEnabled bool) error {
	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO device_status
		(device, serial_number, model, last_seen, is_mounted, 
		 smart_enabled, last_smart_check)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, device, serial, model, time.Now(), isMounted, smartEnabled, time.Now())

	return err
}

// checkHealthThresholds checks for potential health issues and generates alerts
func (m *MAIDSmartMonitor) checkHealthThresholds(attributes []map[string]interface{}) {
	criticalAttrs := map[int]bool{5: true, 187: true, 196: true, 197: true, 198: true}

	for _, attr := range attributes {
		attrID := attr["attribute_id"].(int)
		device := attr["device"].(string)
		rawValue := attr["raw_value"].(int64)
		normalizedValue := attr["normalized_value"].(int)
		threshold := attr["threshold"].(int)
		attrName := attr["attribute_name"].(string)

		// Check for threshold violations
		if threshold > 0 && normalizedValue <= threshold {
			m.createAlert(device, attrName, "THRESHOLD_VIOLATION",
				fmt.Sprintf("Value %d below threshold %d", normalizedValue, threshold))
		}

		// Check critical attributes
		if criticalAttrs[attrID] && rawValue > 0 {
			m.createAlert(device, attrName, "CRITICAL_VALUE",
				fmt.Sprintf("Non-zero critical value: %d", rawValue))
		}

		// Temperature warnings
		if (attrID == 190 || attrID == 194) && rawValue > 60 {
			m.createAlert(device, attrName, "HIGH_TEMPERATURE",
				fmt.Sprintf("High temperature: %d°C", rawValue))
		}
	}
}

// createAlert creates health alert in database
func (m *MAIDSmartMonitor) createAlert(device, attribute, alertType, message string) {
	_, err := m.db.Exec(`
		INSERT INTO health_alerts 
		(device, attribute_name, alert_type, message, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, device, attribute, alertType, message, time.Now())

	if err != nil {
		m.logger.Printf("Failed to create alert: %v", err)
	} else {
		m.logger.Printf("HEALTH ALERT - %s: %s - %s", device, attribute, message)
	}
}

// runMonitoringCycle runs a single monitoring cycle
func (m *MAIDSmartMonitor) runMonitoringCycle() error {
	m.logger.Println("Starting SMART monitoring cycle...")

	mountedDrives, err := m.getMountedDrives()
	if err != nil {
		return fmt.Errorf("failed to get mounted drives: %v", err)
	}

	for _, device := range mountedDrives {
		m.logger.Printf("Processing device: %s", device)

		// Get device info without spinning up
		serial, model, err := m.getDeviceInfo(device)
		if err != nil {
			m.logger.Printf("Failed to get device info for %s: %v", device, err)
			continue
		}

		smartEnabled := m.checkSmartSupport(device)

		// Update device status
		if err := m.updateDeviceStatus(device, serial, model, true, smartEnabled); err != nil {
			m.logger.Printf("Failed to update device status for %s: %v", device, err)
		}

		if !smartEnabled {
			m.logger.Printf("SMART not supported/enabled on %s", device)
			continue
		}

		// Collect SMART data (only if device is already spinning)
		smartData, err := m.collectSmartData(device)
		if err != nil {
			m.logger.Printf("Error collecting SMART data for %s: %v", device, err)
			continue
		}

		if smartData != nil {
			attributes := m.parseSmartAttributes(smartData, device)
			if len(attributes) > 0 {
				if err := m.storeSmartData(attributes, serial, model); err != nil {
					m.logger.Printf("Failed to store SMART data for %s: %v", device, err)
				} else {
					m.checkHealthThresholds(attributes)
				}
			} else {
				m.logger.Printf("No target SMART attributes found for %s", device)
			}
		} else {
			m.logger.Printf("No SMART data collected for %s (likely in standby)", device)
		}
	}

	m.logger.Println("Monitoring cycle completed")
	return nil
}

// getHealthSummary gets health summary from database
func (m *MAIDSmartMonitor) getHealthSummary() (map[string]interface{}, error) {
	// Get alerts by device
	rows, err := m.db.Query(`
		SELECT device, COUNT(*) as alert_count
		FROM health_alerts 
		WHERE resolved = FALSE 
		GROUP BY device
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %v", err)
	}
	defer rows.Close()

	alertsByDevice := make(map[string]int)
	for rows.Next() {
		var device string
		var count int
		if err := rows.Scan(&device, &count); err != nil {
			return nil, fmt.Errorf("failed to scan alert row: %v", err)
		}
		alertsByDevice[device] = count
	}

	// Get device count
	var deviceCount int
	err = m.db.QueryRow("SELECT COUNT(DISTINCT device) FROM device_status").Scan(&deviceCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get device count: %v", err)
	}

	return map[string]interface{}{
		"total_devices":       deviceCount,
		"devices_with_alerts": len(alertsByDevice),
		"alerts_by_device":    alertsByDevice,
	}, nil
}

// exportData exports SMART data to CSV for analysis
func (m *MAIDSmartMonitor) exportData(outputFile string, days int) error {
	rows, err := m.db.Query(`
		SELECT * FROM smart_data 
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		ORDER BY device, timestamp, attribute_id
	`, days)
	if err != nil {
		return fmt.Errorf("failed to query data: %v", err)
	}
	defer rows.Close()

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}
	writer.Write(columns)

	// Write data
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		record := make([]string, len(columns))
		for i, val := range values {
			if val != nil {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		writer.Write(record)
	}

	m.logger.Printf("Data exported to %s", outputFile)
	return nil
}

func main() {
	var (
		dbPath   = flag.String("db", "maid_smart_data.db", "Database file path")
		interval = flag.Int("interval", 300, "Monitoring interval in seconds")
		daemon   = flag.Bool("daemon", false, "Run as daemon")
		export   = flag.String("export", "", "Export data to CSV file")
		summary  = flag.Bool("summary", false, "Show health summary")
	)
	flag.Parse()

	monitor, err := NewMAIDSmartMonitor(*dbPath)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	if *export != "" {
		if err := monitor.exportData(*export, 30); err != nil {
			log.Fatalf("Failed to export data: %v", err)
		}
		return
	}

	if *summary {
		summary, err := monitor.getHealthSummary()
		if err != nil {
			log.Fatalf("Failed to get health summary: %v", err)
		}

		fmt.Println("MAID SMART Health Summary:")
		fmt.Printf("Total devices: %v\n", summary["total_devices"])
		fmt.Printf("Devices with alerts: %v\n", summary["devices_with_alerts"])

		if alerts, ok := summary["alerts_by_device"].(map[string]int); ok {
			for device, count := range alerts {
				fmt.Printf("  %s: %d alerts\n", device, count)
			}
		}
		return
	}

	if *daemon {
		monitor.logger.Printf("Starting MAID SMART monitor daemon (interval: %ds)", *interval)

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(time.Duration(*interval) * time.Second)
		defer ticker.Stop()

		// Run initial cycle
		if err := monitor.runMonitoringCycle(); err != nil {
			monitor.logger.Printf("Error in monitoring cycle: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := monitor.runMonitoringCycle(); err != nil {
					monitor.logger.Printf("Error in monitoring cycle: %v", err)
				}
			case sig := <-sigChan:
				monitor.logger.Printf("Received signal %v, shutting down...", sig)
				return
			}
		}
	} else {
		if err := monitor.runMonitoringCycle(); err != nil {
			log.Fatalf("Error in monitoring cycle: %v", err)
		}
	}
}
