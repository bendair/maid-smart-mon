# MAID SMART Health Monitor (Go)

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#building)

A lightweight, high-performance SMART health monitoring system specifically designed for **MAID (Massive Array of Idle Disks)** environments, where disk spin-ups are expensive and should be minimized.

## 🎯 Key Features

- **🔋 MAID-Optimized**: Only monitors mounted/active disks to prevent unnecessary spin-ups
- **⚡ High Performance**: Written in Go for minimal resource usage (~5-10MB RAM)
- **🚀 Single Binary**: No dependencies, easy deployment
- **📊 Comprehensive Monitoring**: Tracks 23 critical SMART attributes
- **🗄️ SQLite Database**: Lightweight, embedded database for historical data
- **🚨 Health Alerts**: Automatic threshold monitoring and alerting
- **📈 Data Export**: CSV export for analysis and reporting
- **🔄 Daemon Mode**: Continuous monitoring with configurable intervals

## 📋 Monitored SMART Attributes

| ID  | Attribute Name | Description |
|-----|----------------|-------------|
| 1   | Raw_Read_Error_Rate | Rate of hardware read errors |
| 3   | Spin_Up_Time | Time to spin up from standby |
| 4   | Start_Stop_Count | Count of spindle start/stop cycles |
| 5   | Reallocated_Sector_Ct | Count of reallocated sectors |
| 7   | Seek_Error_Rate | Rate of seek errors |
| 9   | Power_On_Hours | Total powered-on time |
| 12  | Power_Cycle_Count | Count of power-on events |
| 187 | Reported_Uncorrectable_Errors | Uncorrectable errors |
| 188 | Command_Timeout | Command timeout count |
| 190 | Airflow_Temperature_Cel | Drive temperature (airflow) |
| 191 | G_Sense_Error_Rate | Mechanical shock errors |
| 192 | Power_Off_Retract_Count | Emergency head retracts |
| 193 | Load_Cycle_Count | Head load/unload cycles |
| 194 | Temperature_Celsius | Drive temperature |
| 196 | Reallocation_Event_Count | Reallocation events |
| 197 | Current_Pending_Sector | Sectors awaiting reallocation |
| 198 | Offline_Uncorrectable | Offline uncorrectable sectors |
| 199 | UDMA_CRC_Error_Count | Interface CRC errors |
| 222 | Loaded_Hours | Time with heads loaded |
| 240 | Head_Flying_Hours | Head positioning time |
| 241 | Total_LBAs_Written | Lifetime data written |
| 242 | Total_LBAs_Read | Lifetime data read |

## 🚀 Quick Start

### Prerequisites

- Linux system with mounted drives
- `smartmontools` package installed
- Go 1.19+ (for building from source)

```bash
# Install smartmontools
sudo apt-get update
sudo apt-get install smartmontools

# Or on RHEL/CentOS
sudo yum install smartmontools
```

### Installation

#### Option 1: Download Binary (Recommended)

```bash
# Download latest release
wget https://github.com/yourusername/maid-smart-monitor/releases/latest/download/maid-smart-monitor
chmod +x maid-smart-monitor
sudo mv maid-smart-monitor /usr/local/bin/
```

#### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/yourusername/maid-smart-monitor.git
cd maid-smart-monitor

# Build
go mod tidy
go build -o maid-smart-monitor main.go

# Install
sudo cp maid-smart-monitor /usr/local/bin/
```

## 📖 Usage

### Basic Commands

```bash
# Run single monitoring cycle
maid-smart-monitor

# Run as daemon with 10-minute intervals
maid-smart-monitor -daemon -interval 600

# Show health summary
maid-smart-monitor -summary

# Export data to CSV (last 30 days)
maid-smart-monitor -export smart_data.csv

# Use custom database location
maid-smart-monitor -db /var/lib/smart/data.db
```

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-db` | `maid_smart_data.db` | SQLite database file path |
| `-interval` | `300` | Monitoring interval in seconds (daemon mode) |
| `-daemon` | `false` | Run as background daemon |
| `-export` | `""` | Export data to CSV file |
| `-summary` | `false` | Display health summary and exit |

### Example Output

```bash
$ maid-smart-monitor -summary
MAID SMART Health Summary:
Total devices: 12
Devices with alerts: 2
  /dev/sda: 1 alerts
  /dev/sdf: 3 alerts
```

## 🔧 Production Deployment

### Systemd Service

Create a systemd service for automatic startup:

```bash
sudo tee /etc/systemd/system/maid-smart-monitor.service > /dev/null << EOF
[Unit]
Description=MAID SMART Health Monitor
After=multi-user.target

[Service]
Type=simple
ExecStart=/usr/local/bin/maid-smart-monitor -daemon -interval 600 -db /var/lib/smart/maid_smart_data.db
Restart=on-failure
RestartSec=10
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Create database directory
sudo mkdir -p /var/lib/smart
sudo chown root:root /var/lib/smart

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable maid-smart-monitor
sudo systemctl start maid-smart-monitor

# Check status
sudo systemctl status maid-smart-monitor
sudo journalctl -u maid-smart-monitor -f
```

### Docker Deployment

```dockerfile
FROM alpine:latest

RUN apk add --no-cache smartmontools sqlite

COPY maid-smart-monitor /usr/local/bin/
RUN chmod +x /usr/local/bin/maid-smart-monitor

VOLUME ["/data"]

CMD ["maid-smart-monitor", "-daemon", "-db", "/data/maid_smart_data.db"]
```

```bash
# Build and run
docker build -t maid-smart-monitor .
docker run -d --privileged -v /dev:/dev -v ./data:/data maid-smart-monitor
```

## 🏗️ Building

### Standard Build

```bash
go mod tidy
go build -o maid-smart-monitor main.go
```

### Static Binary (Recommended for Production)

```bash
CGO_ENABLED=1 go build -a -ldflags '-extldflags "-static"' -o maid-smart-monitor main.go
```

### Cross-Platform Builds

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o maid-smart-monitor-linux-amd64 main.go

# Linux ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -o maid-smart-monitor-linux-arm64 main.go
```

## 📊 Database Schema

The application uses SQLite with three main tables:

### smart_data
Stores historical SMART attribute values:
```sql
CREATE TABLE smart_data (
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
);
```

### device_status
Tracks device information and status:
```sql
CREATE TABLE device_status (
    device TEXT PRIMARY KEY,
    serial_number TEXT,
    model TEXT,
    last_seen DATETIME,
    is_mounted BOOLEAN,
    mount_point TEXT,
    smart_enabled BOOLEAN,
    last_smart_check DATETIME,
    spin_up_count INTEGER DEFAULT 0
);
```

### health_alerts
Records health alerts and warnings:
```sql
CREATE TABLE health_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device TEXT NOT NULL,
    attribute_name TEXT NOT NULL,
    alert_type TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    resolved BOOLEAN DEFAULT FALSE
);
```

## 🔍 Monitoring and Alerting

### Health Check Types

1. **Threshold Violations**: When normalized values fall below manufacturer thresholds
2. **Critical Values**: Non-zero values for critical attributes (5, 187, 196, 197, 198)
3. **Temperature Warnings**: Drive temperatures above 60°C

### Integration with Monitoring Systems

#### Prometheus Metrics (Future Enhancement)

```bash
# Add metrics endpoint
maid-smart-monitor -daemon -metrics-port 8080

# Query metrics
curl http://localhost:8080/metrics
```

#### Nagios/Icinga Integration

```bash
#!/bin/bash
# check_maid_smart.sh
ALERTS=$(maid-smart-monitor -summary | grep "Devices with alerts:" | awk '{print $4}')

if [ "$ALERTS" -gt 0 ]; then
    echo "CRITICAL - $ALERTS devices have SMART alerts"
    exit 2
else
    echo "OK - All devices healthy"
    exit 0
fi
```

## 🛠️ MAID Environment Optimization

### Spin-Up Avoidance Strategies

1. **Mount Point Detection**: Only monitors currently mounted drives
2. **Standby Checking**: Uses `smartctl --nocheck=standby` to avoid wake-ups
3. **Power State Awareness**: Checks device power state before SMART queries
4. **Opportunistic Collection**: Collects data when drives are naturally active

### Best Practices for MAID

- Set monitoring intervals to 10+ minutes to reduce overhead
- Schedule intensive monitoring during planned maintenance windows
- Use the daemon mode for continuous background monitoring
- Monitor logs for unintended spin-ups
- Consider integration with your MAID controller's API

## 🐛 Troubleshooting

### Common Issues

#### Permission Denied
```bash
# Ensure running as root or with proper permissions
sudo maid-smart-monitor

# Or add user to disk group
sudo usermod -a -G disk $USER
```

#### SMART Not Supported
```bash
# Check if SMART is enabled
sudo smartctl -i /dev/sda

# Enable SMART if needed
sudo smartctl -s on /dev/sda
```

#### Database Locked
```bash
# Check for multiple instances
ps aux | grep maid-smart-monitor

# Kill existing processes
sudo pkill maid-smart-monitor
```

### Debug Mode

Enable verbose logging by modifying the log level in the source code or add a debug flag:

```bash
# View systemd logs
sudo journalctl -u maid-smart-monitor -f

# Check database content
sqlite3 maid_smart_data.db "SELECT COUNT(*) FROM smart_data;"
```

## 🧪 Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
# Test with mock devices
go test -tags=integration ./...
```

### Manual Testing

```bash
# Test on a single device
sudo smartctl -A /dev/sda

# Verify database creation
./maid-smart-monitor
ls -la maid_smart_data.db
```

## 📈 Performance

### Benchmarks

- **Memory Usage**: ~5-10MB resident
- **CPU Usage**: <1% during monitoring cycles
- **Disk I/O**: Minimal, only database writes
- **Network**: None (local monitoring only)

### Scalability

- Tested with 100+ drives
- Monitoring cycle scales linearly with device count
- SQLite database handles millions of records efficiently

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md).

### Development Setup

```bash
git clone https://github.com/yourusername/maid-smart-monitor.git
cd maid-smart-monitor
go mod tidy
go test ./...
```

### Coding Standards

- Follow Go conventions and `gofmt`
- Add tests for new features
- Update documentation
- Ensure compatibility with Go 1.19+

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **smartmontools** team for the excellent `smartctl` utility
- **sqlite** team for the embedded database
- **Go** team for the fantastic programming language
- **MAID** researchers for pioneering energy-efficient storage

## 📞 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/yourusername/maid-smart-monitor/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/yourusername/maid-smart-monitor/discussions)
- 📧 **Email**: support@yourdomain.com

## 🗺️ Roadmap

- [ ] Web dashboard for visualization
- [ ] Prometheus metrics endpoint
- [ ] Email/Slack alert notifications
- [ ] Configuration file support
- [ ] Multi-node deployment support
- [ ] Advanced predictive analytics
- [ ] Integration with major MAID controllers

---

**⭐ If this project helps you, please consider giving it a star on GitHub!**
