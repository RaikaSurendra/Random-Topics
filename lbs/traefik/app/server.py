#!/usr/bin/env python3
"""
Simple Flask application for demonstrating load balancing.
Each instance reports its hostname, IP, and request details.
"""

import os
import socket
import datetime
import logging
from flask import Flask, request, jsonify

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | Instance: %(hostname)s | %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)

class HostnameFilter(logging.Filter):
    def filter(self, record):
        record.hostname = socket.gethostname()
        return True

logger = logging.getLogger(__name__)
logger.addFilter(HostnameFilter())

app = Flask(__name__)

# Track request count per instance
request_count = 0

def get_instance_info():
    """Gather instance information for response."""
    return {
        "hostname": socket.gethostname(),
        "ip_address": socket.gethostbyname(socket.gethostname()),
        "instance_id": os.environ.get("INSTANCE_ID", "unknown"),
        "instance_color": os.environ.get("INSTANCE_COLOR", "#ffffff"),
    }

@app.route("/")
def home():
    """Main endpoint - shows instance info and request details."""
    global request_count
    request_count += 1

    instance = get_instance_info()

    response_data = {
        "message": "Hello from Traefik Learning Project!",
        "instance": instance,
        "request": {
            "number": request_count,
            "method": request.method,
            "path": request.path,
            "remote_addr": request.remote_addr,
            "headers": {
                "X-Forwarded-For": request.headers.get("X-Forwarded-For", "N/A"),
                "X-Real-IP": request.headers.get("X-Real-IP", "N/A"),
                "Host": request.headers.get("Host", "N/A"),
            }
        },
        "timestamp": datetime.datetime.now().isoformat()
    }

    logger.info(f"Request #{request_count} from {request.remote_addr} to {request.path}")

    return jsonify(response_data)

@app.route("/health")
def health():
    """Health check endpoint for load balancer."""
    instance = get_instance_info()
    logger.info("Health check passed")
    return jsonify({
        "status": "healthy",
        "instance": instance["hostname"],
        "timestamp": datetime.datetime.now().isoformat()
    })

@app.route("/heavy")
def heavy_task():
    """Simulate a heavy task for load testing."""
    import time
    global request_count
    request_count += 1

    instance = get_instance_info()

    # Simulate processing time
    process_time = 0.5
    time.sleep(process_time)

    logger.info(f"Heavy task completed in {process_time}s")

    return jsonify({
        "message": "Heavy task completed",
        "instance": instance,
        "processing_time_seconds": process_time,
        "request_number": request_count
    })

@app.route("/info")
def info():
    """Detailed instance information."""
    instance = get_instance_info()

    return jsonify({
        "instance": instance,
        "environment": {
            "python_version": os.popen("python --version").read().strip(),
            "total_requests_served": request_count,
        },
        "network": {
            "hostname": socket.gethostname(),
            "fqdn": socket.getfqdn(),
        }
    })

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5000))
    logger.info(f"Starting server on port {port}")
    app.run(host="0.0.0.0", port=port, debug=False)
