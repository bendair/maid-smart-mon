#!/usr/bin/env python3
"""
MAID SMART Health Monitoring System
Designed for Massive Array of Idle Disks environments where disk spin-ups are expensive.
Only monitors currently mounted/active disks to avoid unnecessary spin-ups.
"""

import sqlite3
import subprocess
import json
import re
import time
import logging
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Tuple
import argparse
import psutil

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('maid_smart_monitor.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

class MAIDSmartMonitor:
    """SMART monitoring system optimized for MAID environments"""
    
    # Target SMART attributes for monitoring
    TARGET_ATTRIBUTES = {
        1: 'Raw_Read_Error_Rate',
        3: 'Spin_Up_Time', 
        4: 'Start_Stop_Count',
        5: 'Reallocated_Sector_Ct',
        7: 'Seek_Error_Rate',
        9: 'Power_On_Hours',
        12: 'Power_Cycle_Count',
        187: 'Reported_Uncorrectable_Errors',
        188: 'Command_Timeout',
        190: 'Airflow_Temperature_Cel',
        191: 'G_Sense_Error_Rate',
        192: 'Power_Off_Retract_Count',
        193: 'Load_Cycle_Count',
        194: 'Temperature_Celsius',
        196: 'Reallocation_Event_Count',
        197: 'Current_Pending_Sector',
        198: 'Offline_Uncorrectable',
        199: 'UDMA_CRC_Error_Count',
        222: 'Loaded_Hours',
        240: 'Head_Flying_Hours',
        241: 'Total_LBAs_Written',
        242: 'Total_LBAs_Read'
    }
    
    def __init__(self, db_path: str = "maid_smart_data.db"):
        self.db_path = db_path
        self.init_database()
        
    def init_database(self):
        """Initialize SQLite database with required tables"""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            # Main SMART data table
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS smart_data (
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
                )
            ''')
            
            # Device status table to track spin state
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS device_status (
                    device TEXT PRIMARY KEY,
                    serial_number TEXT,
                    model TEXT,
                    last_seen DATETIME,
                    is_mounted BOOLEAN,
                    mount_point TEXT,
                    smart_enabled BOOLEAN,
                    last_smart_check DATETIME,
                    spin_up_count INTEGER DEFAULT 0
                )
            ''')
            
            # Health alerts table
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS health_alerts (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    device TEXT NOT NULL,
                    attribute_name TEXT NOT NULL,
                    alert_type TEXT NOT NULL,
                    message TEXT NOT NULL,
                    timestamp DATETIME NOT NULL,
                    resolved BOOLEAN DEFAULT FALSE
                )
            ''')
            
            conn.commit()
            logger.info(f"Database initialized: {self.db_path}")
    
    def get_mounted_drives(self) -> List[str]:
        """Get list of currently mounted drives to avoid spinning up idle disks"""
        mounted_drives = []
        
        # Get mounted partitions
        partitions = psutil.disk_partitions()
        
        for partition in partitions:
            # Extract device name (e.g., /dev/sda1 -> /dev/sda)
            device = re.sub(r'\d+$', '', partition.device)
            if device not in mounted_drives and device.startswith('/dev/sd'):
                mounted_drives.append(device)
        
        logger.info(f"Found {len(mounted_drives)} mounted drives: {mounted_drives}")
        return mounted_drives
    
    def check_smart_support(self, device: str) -> bool:
        """Check if device supports SMART without spinning it up"""
        try:
            # Use --nocheck=standby to avoid spinning up idle drives
            result = subprocess.run([
                'smartctl', '--nocheck=standby', '-i', device
            ], capture_output=True, text=True, timeout=30)
            
            if result.returncode == 0:
                return 'SMART support is: Enabled' in result.stdout
            else:
                logger.warning(f"SMART support check failed for {device}: {result.stderr}")
                return False
                
        except subprocess.TimeoutExpired:
            logger.error(f"Timeout checking SMART support for {device}")
            return False
        except Exception as e:
            logger.error(f"Error checking SMART support for {device}: {e}")
            return False
    
    def get_device_info(self, device: str) -> Tuple[Optional[str], Optional[str]]:
        """Get device serial number and model without spinning up"""
        try:
            result = subprocess.run([
                'smartctl', '--nocheck=standby', '-i', device
            ], capture_output=True, text=True, timeout=30)
            
            if result.returncode == 0:
                serial = None
                model = None
                
                for line in result.stdout.split('\n'):
                    if 'Serial Number:' in line:
                        serial = line.split('Serial Number:')[1].strip()
                    elif 'Device Model:' in line or 'Model Number:' in line:
                        model = line.split(':')[1].strip()
                
                return serial, model
            
        except Exception as e:
            logger.error(f"Error getting device info for {device}: {e}")
        
        return None, None
    
    def collect_smart_data(self, device: str) -> Dict:
        """Collect SMART data from a device (only if already spinning)"""
        try:
            # First check if device is in standby mode
            standby_check = subprocess.run([
                'smartctl', '--nocheck=standby', '-n', 'standby', device
            ], capture_output=True, text=True, timeout=10)
            
            if 'STANDBY' in standby_check.stdout:
                logger.info(f"Device {device} is in standby mode - skipping to avoid spin-up")
                return {}
            
            # Device is already spinning, safe to collect SMART data
            result = subprocess.run([
                'smartctl', '-A', '--json', device
            ], capture_output=True, text=True, timeout=60)
            
            if result.returncode == 0:
                data = json.loads(result.stdout)
                return data
            else:
                logger.warning(f"SMART data collection failed for {device}: {result.stderr}")
                return {}
                
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse SMART JSON for {device}: {e}")
            return {}
        except subprocess.TimeoutExpired:
            logger.error(f"Timeout collecting SMART data for {device}")
            return {}
        except Exception as e:
            logger.error(f"Error collecting SMART data for {device}: {e}")
            return {}
    
    def parse_smart_attributes(self, smart_data: Dict, device: str) -> List[Dict]:
        """Parse and filter SMART attributes for target IDs"""
        attributes = []
        
        if 'ata_smart_attributes' not in smart_data:
            return attributes
        
        table = smart_data['ata_smart_attributes'].get('table', [])
        
        for attr in table:
            attr_id = attr.get('id')
            if attr_id in self.TARGET_ATTRIBUTES:
                attributes.append({
                    'device': device,
                    'attribute_id': attr_id,
                    'attribute_name': self.TARGET_ATTRIBUTES[attr_id],
                    'raw_value': attr.get('raw', {}).get('value', 0),
                    'normalized_value': attr.get('value', 0),
                    'threshold': attr.get('thresh', 0),
                    'worst_value': attr.get('worst', 0),
                    'flags': str(attr.get('flags', {}))
                })
        
        return attributes
    
    def store_smart_data(self, attributes: List[Dict], serial: str, model: str):
        """Store SMART attributes in database"""
        if not attributes:
            return
        
        timestamp = datetime.now()
        
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            for attr in attributes:
                cursor.execute('''
                    INSERT OR REPLACE INTO smart_data 
                    (device, serial_number, model, timestamp, attribute_id, attribute_name,
                     raw_value, normalized_value, threshold, worst_value, flags)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''', (
                    attr['device'], serial, model, timestamp,
                    attr['attribute_id'], attr['attribute_name'],
                    attr['raw_value'], attr['normalized_value'], 
                    attr['threshold'], attr['worst_value'], attr['flags']
                ))
            
            conn.commit()
            logger.info(f"Stored {len(attributes)} SMART attributes for {attributes[0]['device']}")
    
    def update_device_status(self, device: str, serial: str, model: str, 
                           is_mounted: bool, smart_enabled: bool):
        """Update device status in database"""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            cursor.execute('''
                INSERT OR REPLACE INTO device_status
                (device, serial_number, model, last_seen, is_mounted, 
                 smart_enabled, last_smart_check)
                VALUES (?, ?, ?, ?, ?, ?, ?)
            ''', (device, serial, model, datetime.now(), is_mounted, 
                  smart_enabled, datetime.now()))
            
            conn.commit()
    
    def check_health_thresholds(self, attributes: List[Dict]):
        """Check for potential health issues and generate alerts"""
        critical_attrs = [5, 187, 196, 197, 198]  # Critical health indicators
        
        for attr in attributes:
            attr_id = attr['attribute_id']
            device = attr['device']
            raw_value = attr['raw_value']
            normalized_value = attr['normalized_value']
            threshold = attr['threshold']
            
            # Check for threshold violations
            if threshold > 0 and normalized_value <= threshold:
                self.create_alert(device, attr['attribute_name'], 'THRESHOLD_VIOLATION',
                                f"Value {normalized_value} below threshold {threshold}")
            
            # Check critical attributes
            if attr_id in critical_attrs and raw_value > 0:
                self.create_alert(device, attr['attribute_name'], 'CRITICAL_VALUE',
                                f"Non-zero critical value: {raw_value}")
            
            # Temperature warnings
            if attr_id in [190, 194] and raw_value > 60:  # Temperature > 60°C
                self.create_alert(device, attr['attribute_name'], 'HIGH_TEMPERATURE',
                                f"High temperature: {raw_value}°C")
    
    def create_alert(self, device: str, attribute: str, alert_type: str, message: str):
        """Create health alert in database"""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            cursor.execute('''
                INSERT INTO health_alerts 
                (device, attribute_name, alert_type, message, timestamp)
                VALUES (?, ?, ?, ?, ?)
            ''', (device, attribute, alert_type, message, datetime.now()))
            
            conn.commit()
            logger.warning(f"HEALTH ALERT - {device}: {attribute} - {message}")
    
    def run_monitoring_cycle(self):
        """Run a single monitoring cycle"""
        logger.info("Starting SMART monitoring cycle...")
        
        mounted_drives = self.get_mounted_drives()
        
        for device in mounted_drives:
            try:
                logger.info(f"Processing device: {device}")
                
                # Get device info without spinning up
                serial, model = self.get_device_info(device)
                smart_enabled = self.check_smart_support(device)
                
                # Update device status
                self.update_device_status(device, serial, model, True, smart_enabled)
                
                if not smart_enabled:
                    logger.warning(f"SMART not supported/enabled on {device}")
                    continue
                
                # Collect SMART data (only if device is already spinning)
                smart_data = self.collect_smart_data(device)
                
                if smart_data:
                    attributes = self.parse_smart_attributes(smart_data, device)
                    if attributes:
                        self.store_smart_data(attributes, serial, model)
                        self.check_health_thresholds(attributes)
                    else:
                        logger.info(f"No target SMART attributes found for {device}")
                else:
                    logger.info(f"No SMART data collected for {device} (likely in standby)")
                    
            except Exception as e:
                logger.error(f"Error processing device {device}: {e}")
        
        logger.info("Monitoring cycle completed")
    
    def get_health_summary(self) -> Dict:
        """Get health summary from database"""
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            # Get latest data for each device
            cursor.execute('''
                SELECT device, COUNT(*) as alert_count
                FROM health_alerts 
                WHERE resolved = FALSE 
                GROUP BY device
            ''')
            
            alerts = dict(cursor.fetchall())
            
            # Get device count
            cursor.execute('SELECT COUNT(DISTINCT device) FROM device_status')
            device_count = cursor.fetchone()[0]
            
            return {
                'total_devices': device_count,
                'devices_with_alerts': len(alerts),
                'alerts_by_device': alerts
            }
    
    def export_data(self, output_file: str, days: int = 30):
        """Export SMART data to CSV for analysis"""
        import csv
        
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.cursor()
            
            cursor.execute('''
                SELECT * FROM smart_data 
                WHERE timestamp >= datetime('now', '-{} days')
                ORDER BY device, timestamp, attribute_id
            '''.format(days))
            
            with open(output_file, 'w', newline='') as csvfile:
                writer = csv.writer(csvfile)
                writer.writerow([desc[0] for desc in cursor.description])
                writer.writerows(cursor.fetchall())
        
        logger.info(f"Data exported to {output_file}")

def main():
    parser = argparse.ArgumentParser(description='MAID SMART Health Monitor')
    parser.add_argument('--db', default='maid_smart_data.db', 
                       help='Database file path')
    parser.add_argument('--interval', type=int, default=300,
                       help='Monitoring interval in seconds (default: 300)')
    parser.add_argument('--daemon', action='store_true',
                       help='Run as daemon')
    parser.add_argument('--export', type=str,
                       help='Export data to CSV file')
    parser.add_argument('--summary', action='store_true',
                       help='Show health summary')
    
    args = parser.parse_args()
    
    monitor = MAIDSmartMonitor(args.db)
    
    if args.export:
        monitor.export_data(args.export)
        return
    
    if args.summary:
        summary = monitor.get_health_summary()
        print(f"MAID SMART Health Summary:")
        print(f"Total devices: {summary['total_devices']}")
        print(f"Devices with alerts: {summary['devices_with_alerts']}")
        for device, count in summary['alerts_by_device'].items():
            print(f"  {device}: {count} alerts")
        return
    
    if args.daemon:
        logger.info(f"Starting MAID SMART monitor daemon (interval: {args.interval}s)")
        try:
            while True:
                monitor.run_monitoring_cycle()
                time.sleep(args.interval)
        except KeyboardInterrupt:
            logger.info("Daemon stopped by user")
    else:
        monitor.run_monitoring_cycle()

if __name__ == '__main__':
    main()
