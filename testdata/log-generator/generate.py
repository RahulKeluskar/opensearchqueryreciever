#!/usr/bin/env python3
"""
Log Generator for OpenSearch

Generates synthetic log data and indexes it into OpenSearch
for testing the OpenSearch Query Receiver.
"""

import os
import sys
import time
import random
import json
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Any
import requests
from requests.auth import HTTPBasicAuth

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration from environment
OPENSEARCH_ENDPOINT = os.getenv('OPENSEARCH_ENDPOINT', 'https://localhost:9200')
OPENSEARCH_USERNAME = os.getenv('OPENSEARCH_USERNAME', 'admin')
OPENSEARCH_PASSWORD = os.getenv('OPENSEARCH_PASSWORD', 'admin')
GENERATION_INTERVAL = int(os.getenv('GENERATION_INTERVAL', '10'))  # seconds
LOGS_PER_BATCH = int(os.getenv('LOGS_PER_BATCH', '50'))

# Disable SSL warnings for self-signed certificates
requests.packages.urllib3.disable_warnings()

# Sample data for log generation
LOG_LEVELS = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL']
LOG_LEVEL_WEIGHTS = [5, 60, 20, 10, 5]  # Probability weights

SERVICES = ['api-gateway', 'auth-service', 'payment-service', 'user-service', 'notification-service']
ENVIRONMENTS = ['development', 'staging', 'production']

MESSAGES = {
    'DEBUG': [
        'Debug information logged',
        'Variable value: {}',
        'Function entry point',
        'Processing request',
    ],
    'INFO': [
        'Request processed successfully',
        'User login successful',
        'Transaction completed',
        'Cache hit',
        'Configuration loaded',
        'Service started',
        'Health check passed',
    ],
    'WARN': [
        'High memory usage detected',
        'Slow query detected',
        'Rate limit approaching',
        'Deprecated API used',
        'Connection pool exhausted',
        'Retry attempt {}',
    ],
    'ERROR': [
        'Database connection failed',
        'Authentication failed',
        'Invalid request parameters',
        'Timeout waiting for response',
        'Payment processing failed',
        'API endpoint not found',
        'Validation error',
    ],
    'CRITICAL': [
        'Database unavailable',
        'Service crashed',
        'Critical security breach detected',
        'Disk space exhausted',
        'Memory allocation failed',
    ]
}

ERROR_CODES = {
    'ERROR': ['ERR_001', 'ERR_002', 'ERR_003', 'ERR_404', 'ERR_500', 'ERR_503'],
    'CRITICAL': ['CRIT_001', 'CRIT_002', 'CRIT_DB_DOWN', 'CRIT_MEM', 'CRIT_DISK']
}

def create_index_template():
    """
    Create an index template for logs
    """
    template = {
        "index_patterns": ["logs-*"],
        "template": {
            "settings": {
                "number_of_shards": 1,
                "number_of_replicas": 0,
                "refresh_interval": "5s"
            },
            "mappings": {
                "properties": {
                    "@timestamp": {"type": "date"},
                    "level": {"type": "keyword"},
                    "message": {"type": "text"},
                    "service": {"type": "keyword"},
                    "environment": {"type": "keyword"},
                    "host": {"type": "keyword"},
                    "user_id": {"type": "keyword"},
                    "request_id": {"type": "keyword"},
                    "response_time_ms": {"type": "integer"},
                    "status_code": {"type": "integer"},
                    "error_code": {"type": "keyword"},
                    "severity": {"type": "keyword"},
                    "transaction_type": {"type": "keyword"},
                    "path": {"type": "keyword"}
                }
            }
        }
    }

    url = f"{OPENSEARCH_ENDPOINT}/_index_template/logs-template"
    auth = HTTPBasicAuth(OPENSEARCH_USERNAME, OPENSEARCH_PASSWORD)

    try:
        response = requests.put(
            url,
            json=template,
            auth=auth,
            verify=False,
            timeout=10
        )
        if response.status_code in [200, 201]:
            logger.info("Index template created successfully")
            return True
        else:
            logger.error(f"Failed to create index template: {response.status_code} - {response.text}")
            return False
    except Exception as e:
        logger.error(f"Error creating index template: {e}")
        return False

def generate_log_entry() -> Dict[str, Any]:
    """
    Generate a single synthetic log entry
    """
    level = random.choices(LOG_LEVELS, weights=LOG_LEVEL_WEIGHTS)[0]
    service = random.choice(SERVICES)
    environment = random.choice(ENVIRONMENTS)

    # Base log entry
    log = {
        '@timestamp': datetime.utcnow().isoformat() + 'Z',
        'level': level,
        'message': random.choice(MESSAGES[level]),
        'service': service,
        'environment': environment,
        'host': f"{service}-{random.randint(1, 5)}",
    }

    # Add user_id for some logs (simulating authenticated requests)
    if random.random() > 0.3:
        log['user_id'] = f"user_{random.randint(1000, 9999)}"

    # Add request_id for all logs
    log['request_id'] = f"req_{random.randint(100000, 999999)}"

    # Add response time for INFO and above
    if level in ['INFO', 'WARN', 'ERROR', 'CRITICAL']:
        log['response_time_ms'] = random.randint(10, 5000)

    # Add status code for some logs
    if level in ['INFO', 'WARN', 'ERROR']:
        if level == 'INFO':
            log['status_code'] = random.choice([200, 201, 204])
        elif level == 'WARN':
            log['status_code'] = random.choice([400, 401, 403, 429])
        else:  # ERROR
            log['status_code'] = random.choice([500, 502, 503, 504])

    # Add error code for errors
    if level in ERROR_CODES:
        log['error_code'] = random.choice(ERROR_CODES[level])

    # Add severity for errors
    if level in ['ERROR', 'CRITICAL']:
        log['severity'] = random.choice(['low', 'medium', 'high', 'critical'])

    # Add transaction type for payment service
    if service == 'payment-service' and random.random() > 0.5:
        log['transaction_type'] = random.choice(['payment', 'refund', 'chargeback'])
        if level == 'ERROR':
            log['status'] = 'failed'
        else:
            log['status'] = 'success'

    # Add path for API logs
    if service == 'api-gateway':
        endpoints = ['/api/v1/users', '/api/v1/products', '/api/v1/orders', '/api/v1/payments']
        log['path'] = random.choice(endpoints)

    return log

def index_logs(logs: List[Dict[str, Any]]) -> bool:
    """
    Bulk index logs into OpenSearch
    """
    # Create today's index name
    index_name = f"logs-{datetime.utcnow().strftime('%Y.%m.%d')}"

    # Build bulk request
    bulk_data = []
    for log in logs:
        # Index action
        bulk_data.append(json.dumps({"index": {"_index": index_name}}))
        # Document
        bulk_data.append(json.dumps(log))

    bulk_body = '\n'.join(bulk_data) + '\n'

    url = f"{OPENSEARCH_ENDPOINT}/_bulk"
    auth = HTTPBasicAuth(OPENSEARCH_USERNAME, OPENSEARCH_PASSWORD)

    try:
        response = requests.post(
            url,
            data=bulk_body,
            headers={'Content-Type': 'application/x-ndjson'},
            auth=auth,
            verify=False,
            timeout=10
        )

        if response.status_code == 200:
            result = response.json()
            if result.get('errors'):
                logger.error("Some documents failed to index")
                return False
            logger.info(f"Indexed {len(logs)} log entries to {index_name}")
            return True
        else:
            logger.error(f"Bulk index failed: {response.status_code} - {response.text}")
            return False

    except Exception as e:
        logger.error(f"Error indexing logs: {e}")
        return False

def wait_for_opensearch():
    """
    Wait for OpenSearch to be ready
    """
    url = f"{OPENSEARCH_ENDPOINT}/_cluster/health"
    auth = HTTPBasicAuth(OPENSEARCH_USERNAME, OPENSEARCH_PASSWORD)

    logger.info("Waiting for OpenSearch to be ready...")
    max_retries = 30
    retry_interval = 10

    for i in range(max_retries):
        try:
            response = requests.get(url, auth=auth, verify=False, timeout=5)
            if response.status_code == 200:
                logger.info("OpenSearch is ready!")
                return True
        except Exception as e:
            logger.debug(f"OpenSearch not ready yet: {e}")

        if i < max_retries - 1:
            logger.info(f"Retrying in {retry_interval} seconds... ({i+1}/{max_retries})")
            time.sleep(retry_interval)

    logger.error("OpenSearch did not become ready in time")
    return False

def main():
    """
    Main log generation loop
    """
    logger.info("Starting log generator")
    logger.info(f"OpenSearch endpoint: {OPENSEARCH_ENDPOINT}")
    logger.info(f"Generation interval: {GENERATION_INTERVAL}s")
    logger.info(f"Logs per batch: {LOGS_PER_BATCH}")

    # Wait for OpenSearch
    if not wait_for_opensearch():
        sys.exit(1)

    # Create index template
    if not create_index_template():
        logger.warning("Failed to create index template, continuing anyway...")

    # Generate logs continuously
    logger.info("Starting log generation loop")
    batch_count = 0

    try:
        while True:
            # Generate batch of logs
            logs = [generate_log_entry() for _ in range(LOGS_PER_BATCH)]

            # Index logs
            if index_logs(logs):
                batch_count += 1
                logger.info(f"Generated batch #{batch_count} ({len(logs)} logs)")
            else:
                logger.error("Failed to index logs")

            # Wait before next batch
            time.sleep(GENERATION_INTERVAL)

    except KeyboardInterrupt:
        logger.info("Stopping log generator")
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()
