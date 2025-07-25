# MAID SMART Health Monitor (Python)

[![Python Version](https://img.shields.io/badge/Python-3.8+-blue.svg)](https://python.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Code Style](https://img.shields.io/badge/Code%20Style-Black-black.svg)](https://github.com/psf/black)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#installation)

A lightweight, intelligent SMART health monitoring system specifically designed for **MAID (Massive Array of Idle Disks)** environments, where disk spin-ups are expensive and should be minimized. Built with Python for maximum flexibility and ease of deployment.

## 🎯 Key Features

- **🔋 MAID-Optimized**: Only monitors mounted/active disks to prevent unnecessary spin-ups
- **🐍 Python Powered**: Easy to customize and extend with rich ecosystem
- **📦 Minimal Dependencies**: Only requires `psutil` and standard library
- **📊 Comprehensive Monitoring**: Tracks 23 critical SMART attributes
- **🗄️ SQLite Database**: Lightweight, embedded database for historical data
- **🚨 Health Alerts**: Automatic threshold monitoring and alerting
- **📈 Data Export**: CSV export for analysis and reporting
- **🔄 Daemon Mode**: Continuous monitoring with configurable intervals
- **🛠️ Extensible**: Easy to integrate with existing Python infrastructure

## 📋 Monitored SMART Attributes

| ID  | Attribute Name | Critical | Description |
|-----|----------------|----------|-------------|
| 1   | Raw_Read_Error_Rate | ⚠️ | Rate of hardware read errors |
| 3   | Spin_Up_Time | ℹ️ | Time to spin up from standby |
| 4   | Start_Stop_Count | ℹ️ | Count of spindle start/stop cycles |
| 5   | Reallocated_Sector_Ct | 🔴 | Count of reallocated sectors |
| 7   | Seek_Error_Rate | ⚠️ | Rate of seek errors |
| 9   | Power_On_Hours | ℹ️ | Total powered-on time |
| 12  | Power_Cycle_Count | ℹ️ | Count of power-on events |
| 187 | Reported_Uncorrectable_Errors | 🔴 | Uncorrectable errors |
| 188 | Command_Timeout | ⚠️ | Command timeout count |
| 190 | Airflow_Temperature_Cel | ⚠️ | Drive temperature (airflow) |
| 191 | G_Sense_Error_Rate | ⚠️ | Mechanical shock errors |
| 192 | Power_Off_Retract_Count | ℹ️ | Emergency head retracts |
| 193 | Load_Cycle_Count | ℹ️ | Head load/unload cycles |
| 194 | Temperature_Celsius | ⚠️ | Drive temperature |
| 196 | Reallocation_Event_Count | 🔴 | Reallocation events |
| 197 | Current_Pending_Sector | 🔴 | Sectors awaiting reallocation |
| 198 | Offline_Uncorrectable | 🔴 | Offline uncorrectable sectors |
| 199 | UDMA_CRC_Error_Count | ⚠️ | Interface CRC errors |
| 222 | Loaded_Hours | ℹ️ | Time with heads loaded |
| 240 | Head_Flying_Hours | ℹ️ | Head positioning time |
| 241 | Total_LBAs_Written | ℹ️ | Lifetime data written |
| 242 | Total_LBAs_Read | ℹ️ | Lifetime data read |

## 🚀 Quick Start

### Prerequisites

- **Python 3.8+**
- **Linux system** with mounted drives
- **smartmontools** package
- **Root privileges** (for disk access)

```bash
# Install system dependencies
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install python3 python3-pip smartmontools

# RHEL/CentOS/Fedora
sudo yum install python3 python3-pip smartmontools

# or
sudo dnf install python3 python3-pip smartmontools
```

### Installation

#### Option 1: pip install (Recommended)

```bash
# Install from PyPI (when published)
pip install maid-smart-monitor

# Or install directly from GitHub
pip install git+https://github.com/yourusername/maid-smart-monitor.git
```

#### Option 2: Manual Installation

```bash
# Clone repository
git clone https://github.com/yourusername/maid-smart-monitor.git
cd maid-smart-monitor

# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Make script executable
chmod +x maid_smart_monitor.py
```

#### Option 3: System-wide Installation

```bash
# Clone and install system-wide
git clone https://github.com/yourusername/maid-smart-monitor.git
cd maid-smart-monitor
sudo cp maid_smart_monitor.py /usr/local/bin/
sudo pip3 install psutil
```

## 📖 Usage

### Basic Commands

```bash
# Run single monitoring cycle
sudo python3 maid_smart_monitor.py

# Run as daemon with 10-minute intervals
sudo python3 maid_smart_monitor.py --daemon --interval 600

# Show health summary
sudo python3 maid_smart_monitor.py --summary

# Export data to CSV (last 30 days)
sudo python3 maid_smart_monitor.py --export smart_data.csv

# Use custom database location
sudo python3 maid_smart_monitor.py --db /var/lib/smart/data.db
```

### Command Line Options

| Argument | Default | Description |
|----------|---------|-------------|
| `--db` | `maid_smart_data.db` | SQLite database file path |
| `--interval` | `300` | Monitoring interval in seconds (daemon mode) |
| `--daemon` | `False` | Run as background daemon |
| `--export` | `None` | Export data to CSV file |
| `--summary` | `False` | Display health summary and exit |

### Example Output

```bash
$ sudo python3 maid_smart_monitor.py --summary
[MAID-SMART] 2025-07-25 10:30:45,123 - INFO - Database initialized: maid_smart_data.db
MAID SMART Health Summary:
Total devices: 12
Devices with alerts: 2
  /dev/sda: 1 alerts
  /dev/sdf: 3 alerts
```

## 🔧 Production Deployment

### Virtual Environment Setup

```bash
# Create dedicated user
sudo useradd -r -s /bin/false maidmonitor

# Create application directory
sudo mkdir -p /opt/maid-smart-monitor
sudo chown maidmonitor:maidmonitor /opt/maid-smart-monitor

# Setup virtual environment
cd /opt/maid-smart-monitor
sudo -u maidmonitor python3 -m venv venv
sudo -u maidmonitor ./venv/bin/pip install psutil

# Copy application
sudo cp maid_smart_monitor.py /opt/maid-smart-monitor/
sudo chown maidmonitor:maidmonitor /opt/maid-smart-monitor/maid_smart_monitor.py
```

### Systemd Service

```bash
sudo tee /etc/systemd/system/maid-smart-monitor.service > /dev/null << 'EOF'
[Unit]
Description=MAID SMART Health Monitor
After=multi-user.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/opt/maid-smart-monitor
Environment=PATH=/opt/maid-smart-monitor/venv/bin
ExecStart=/opt/maid-smart-monitor/venv/bin/python /opt/maid-smart-monitor/maid_smart_monitor.py --daemon --interval 600 --db /var/lib/smart/maid_smart_data.db
Restart=on-failure
RestartSec=10
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

#### Dockerfile

```dockerfile
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && \
    apt-get install -y smartmontools && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY maid_smart_monitor.py .
RUN chmod +x maid_smart_monitor.py

# Create volume for data
VOLUME ["/data"]

# Run as root (required for disk access)
USER root

# Health check
HEALTHCHECK --interval=5m --timeout=10s --start-period=30s \
  CMD python maid_smart_monitor.py --summary || exit 1

CMD ["python", "maid_smart_monitor.py", "--daemon", "--db", "/data/maid_smart_data.db"]
```

#### docker-compose.yml

```yaml
version: '3.8'

services:
  maid-smart-monitor:
    build: .
    container_name: maid-smart-monitor
    restart: unless-stopped
    privileged: true
    volumes:
      - /dev:/dev:ro
      - ./data:/data
      - /proc/mounts:/proc/mounts:ro
    environment:
      - PYTHONUNBUFFERED=1
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

```bash
# Deploy with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f maid-smart-monitor

# Check health
docker-compose exec maid-smart-monitor python maid_smart_monitor.py --summary
```

## 📦 Dependencies

### requirements.txt

```txt
psutil>=5.8.0
```

### Optional Dependencies

```txt
# For advanced features (add to requirements-dev.txt)
black>=22.0.0           # Code formatting
flake8>=4.0.0          # Linting
pytest>=7.0.0          # Testing
pytest-cov>=3.0.0      # Coverage
requests>=2.28.0       # HTTP client for webhook alerts
prometheus-client>=0.14.0  # Metrics export
```

## 🔍 Advanced Configuration

### Environment Variables

```bash
# Configuration via environment variables
export MAID_DB_PATH="/var/lib/smart/maid_smart_data.db"
export MAID_INTERVAL="600"
export MAID_LOG_LEVEL="INFO"
export MAID_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

### Configuration File Support

Create `config.ini`:

```ini
[database]
path = /var/lib/smart/maid_smart_data.db

[monitoring]
interval = 600
mounted_only = true

[alerts]
enable_webhooks = true
webhook_url = https://hooks.slack.com/services/...
email_alerts = false
smtp_server = smtp.gmail.com
smtp_port = 587

[logging]
level = INFO
file = /var/log/maid-smart-monitor.log
```

### Integration with Python Infrastructure

#### Flask Web Interface

```python
# web_interface.py
from flask import Flask, jsonify, render_template
import sqlite3

app = Flask(__name__)

@app.route('/api/health')
def health_summary():
    # Query database and return JSON
    with sqlite3.connect('maid_smart_data.db') as conn:
        cursor = conn.cursor()
        # ... query logic
    return jsonify(results)

@app.route('/dashboard')
def dashboard():
    return render_template('dashboard.html')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
```

#### Prometheus Metrics

```python
# prometheus_exporter.py
from prometheus_client import start_http_server, Gauge
import time

# Define metrics
smart_temperature = Gauge('smart_temperature_celsius', 'Drive temperature', ['device'])
smart_power_hours = Gauge('smart_power_on_hours', 'Power on hours', ['device'])

def update_metrics():
    # Query database and update metrics
    pass

if __name__ == '__main__':
    start_http_server(8000)
    while True:
        update_metrics()
        time.sleep(60)
```

## 🛠️ Development

### Setting up Development Environment

```bash
# Clone repository
git clone https://github.com/yourusername/maid-smart-monitor.git
cd maid-smart-monitor

# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install development dependencies
pip install -r requirements-dev.txt

# Install pre-commit hooks
pre-commit install
```

### Code Quality

```bash
# Format code
black maid_smart_monitor.py

# Lint code
flake8 maid_smart_monitor.py

# Type checking
mypy maid_smart_monitor.py

# Run tests
pytest tests/ -v --cov=maid_smart_monitor
```

### Testing

#### Unit Tests

```python
# tests/test_monitor.py
import unittest
from unittest.mock import patch, MagicMock
from maid_smart_monitor import MAIDSmartMonitor

class TestMAIDSmartMonitor(unittest.TestCase):
    def setUp(self):
        self.monitor = MAIDSmartMonitor(':memory:')
    
    @patch('subprocess.run')
    def test_check_smart_support(self, mock_run):
        mock_run.return_value.returncode = 0
        mock_run.return_value.stdout = "SMART support is: Enabled"
        
        result = self.monitor.check_smart_support('/dev/sda')
        self.assertTrue(result)

if __name__ == '__main__':
    unittest.main()
```

#### Integration Tests

```bash
# Create test environment with loop devices
sudo dd if=/dev/zero of=/tmp/test_disk.img bs=1M count=100
sudo losetup /dev/loop0 /tmp/test_disk.img

# Run integration tests
python3 -m pytest tests/integration/ -v
```

## 📊 Database Schema and Analytics

### Querying Historical Data

```python
# analytics.py
import sqlite3
import pandas as pd
import matplotlib.pyplot as plt

def plot_temperature_trends(device, days=30):
    """Plot temperature trends for a specific device"""
    with sqlite3.connect('maid_smart_data.db') as conn:
        query = """
        SELECT timestamp, raw_value 
        FROM smart_data 
        WHERE device = ? AND attribute_id = 194 
        AND timestamp >= datetime('now', '-{} days')
        ORDER BY timestamp
        """.format(days)
        
        df = pd.read_sql_query(query, conn, params=[device])
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        
        plt.figure(figsize=(12, 6))
        plt.plot(df['timestamp'], df['raw_value'])
        plt.title(f'Temperature Trend - {device}')
        plt.xlabel('Time')
        plt.ylabel('Temperature (°C)')
        plt.xticks(rotation=45)
        plt.tight_layout()
        plt.savefig(f'temp_trend_{device.replace("/", "_")}.png')

def health_report():
    """Generate comprehensive health report"""
    with sqlite3.connect('maid_smart_data.db') as conn:
        # Critical attributes analysis
        critical_query = """
        SELECT device, attribute_name, MAX(raw_value) as max_value,
               COUNT(*) as occurrences
        FROM smart_data 
        WHERE attribute_id IN (5, 187, 196, 197, 198)
        AND raw_value > 0
        GROUP BY device, attribute_name
        ORDER BY max_value DESC
        """
        
        df = pd.read_sql_query(critical_query, conn)
        print("Critical Issues Found:")
        print(df.to_string(index=False))
```

### Performance Monitoring

```python
# performance_monitor.py
import time
import psutil
from contextlib import contextmanager

@contextmanager
def performance_monitor():
    """Monitor performance of monitoring cycles"""
    start_time = time.time()
    start_cpu = psutil.cpu_percent()
    start_memory = psutil.virtual_memory().used
    
    yield
    
    end_time = time.time()
    end_cpu = psutil.cpu_percent()
    end_memory = psutil.virtual_memory().used
    
    print(f"Execution time: {end_time - start_time:.2f}s")
    print(f"CPU usage: {end_cpu - start_cpu:.1f}%")
    print(f"Memory delta: {(end_memory - start_memory) / 1024 / 1024:.1f}MB")

# Usage
with performance_monitor():
    monitor.run_monitoring_cycle()
```

## 🐛 Troubleshooting

### Common Issues

#### Import Errors
```bash
# Install missing dependencies
pip install psutil

# Or reinstall in virtual environment
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

#### Permission Issues
```bash
# Run as root
sudo python3 maid_smart_monitor.py

# Or add user to disk group (limited functionality)
sudo usermod -a -G disk $USER
newgrp disk
```

#### Python Version Compatibility
```bash
# Check Python version
python3 --version

# Install compatible Python version
sudo apt-get install python3.8 python3.8-venv
python3.8 -m venv venv
```

### Debug Mode

Enable debug logging by modifying the script:

```python
# Add at the top of the script
import logging
logging.basicConfig(level=logging.DEBUG)

# Or set environment variable
export PYTHONPATH=/path/to/script
export MAID_DEBUG=1
python3 maid_smart_monitor.py
```

### Log Analysis

```bash
# View real-time logs
sudo journalctl -u maid-smart-monitor -f

# Search for errors
sudo journalctl -u maid-smart-monitor | grep ERROR

# Export logs for analysis
sudo journalctl -u maid-smart-monitor --since="2 days ago" > maid_logs.txt
```

## 📈 Performance Optimization

### Memory Usage

```python
# Memory profiling
import tracemalloc

tracemalloc.start()
monitor.run_monitoring_cycle()
current, peak = tracemalloc.get_traced_memory()
print(f"Current memory usage: {current / 1024 / 1024:.1f}MB")
print(f"Peak memory usage: {peak / 1024 / 1024:.1f}MB")
tracemalloc.stop()
```

### Database Optimization

```sql
-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_smart_device_time ON smart_data(device, timestamp);
CREATE INDEX IF NOT EXISTS idx_smart_attribute ON smart_data(attribute_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_alerts_device ON health_alerts(device, resolved);

-- Vacuum database periodically
VACUUM;

-- Analyze for query optimization
ANALYZE;
```

## 🧪 Testing and Validation

### Automated Testing

```bash
# Run full test suite
python3 -m pytest tests/ -v --cov=maid_smart_monitor --cov-report=html

# Run specific test categories
python3 -m pytest tests/unit/ -v
python3 -m pytest tests/integration/ -v --slow

# Performance benchmarks
python3 -m pytest tests/performance/ -v --benchmark-only
```

### Manual Testing Scenarios

```bash
# Test with various device states
sudo smartctl -s standby /dev/sda  # Put device in standby
python3 maid_smart_monitor.py     # Should not spin up

# Test database corruption recovery
rm maid_smart_data.db
python3 maid_smart_monitor.py     # Should recreate database

# Test with no mounted drives
sudo umount /mnt/data
python3 maid_smart_monitor.py     # Should handle gracefully
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md).

### Development Workflow

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/yourusername/maid-smart-monitor.git`
3. **Create virtual environment**: `python3 -m venv venv && source venv/bin/activate`
4. **Install dev dependencies**: `pip install -r requirements-dev.txt`
5. **Create feature branch**: `git checkout -b feature/your-feature`
6. **Make changes** and add tests
7. **Run tests**: `pytest tests/ -v`
8. **Format code**: `black maid_smart_monitor.py`
9. **Commit changes**: `git commit -m "Description"`
10. **Push and create PR**

### Code Standards

- **PEP 8** compliance (enforced by `black` and `flake8`)
- **Type hints** for function signatures
- **Docstrings** for all public functions
- **Unit tests** for new features
- **Integration tests** for complex workflows

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **smartmontools** team for the excellent `smartctl` utility
- **psutil** developers for system monitoring capabilities
- **Python Software Foundation** for the amazing language
- **SQLite** team for the embedded database
- **MAID** researchers for energy-efficient storage innovations

## 📞 Support and Community

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/yourusername/maid-smart-monitor/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/yourusername/maid-smart-monitor/discussions)
- 📧 **Email**: support@yourdomain.com
- 💬 **Discord**: [Join our Discord](https://discord.gg/yourinvite)
- 📖 **Documentation**: [Full Documentation](https://yourusername.github.io/maid-smart-monitor/)

## 🗺️ Roadmap

### Near Term (v1.1)
- [ ] Configuration file support
- [ ] Email alert notifications
- [ ] Web dashboard (Flask/Django)
- [ ] Prometheus metrics endpoint

### Medium Term (v1.2)
- [ ] Slack/Discord webhook integration
- [ ] Advanced analytics and reporting
- [ ] Multi-node deployment support
- [ ] REST API for external integration

### Long Term (v2.0)
- [ ] Machine learning for predictive analytics
- [ ] Integration with major MAID controllers
- [ ] Real-time streaming dashboard
- [ ] Mobile app for alerts
- [ ] Cloud deployment support (AWS/GCP/Azure)

### Community Requests
- [ ] Windows support
- [ ] NVMe SMART attribute support
- [ ] RAID controller integration
- [ ] Grafana dashboard templates

---

**⭐ If this project helps you, please consider giving it a star on GitHub!**

**🐍 Built with ❤️ in Python for the storage community**
