#!/usr/bin/env python3
"""
OAuth2 Authentication Proxy for OpenSearch

This is a simple proxy that demonstrates how to add OAuth2 authentication
in front of OpenSearch. In production, you would replace this with a proper
OAuth2 provider (e.g., OAuth2 Proxy, Keycloak, etc.)

This proxy:
1. Accepts requests from the OpenTelemetry collector
2. Adds authentication credentials to requests
3. Forwards requests to OpenSearch
4. Returns OpenSearch responses to the client
"""

import os
import requests
import logging
from flask import Flask, request, Response
from urllib.parse import urljoin
from requests.auth import HTTPBasicAuth

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Flask app
app = Flask(__name__)

# Configuration from environment variables
OPENSEARCH_ENDPOINT = os.getenv('OPENSEARCH_ENDPOINT', 'https://localhost:9200')
OPENSEARCH_USERNAME = os.getenv('OPENSEARCH_USERNAME', 'admin')
OPENSEARCH_PASSWORD = os.getenv('OPENSEARCH_PASSWORD', 'admin')

# Disable SSL warnings for self-signed certificates (dev only)
requests.packages.urllib3.disable_warnings()

@app.route('/', defaults={'path': ''}, methods=['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'])
@app.route('/<path:path>', methods=['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'])
def proxy(path):
    """
    Proxy all requests to OpenSearch with authentication
    """
    # Build the target URL
    url = urljoin(OPENSEARCH_ENDPOINT + '/', path)
    if request.query_string:
        url = f"{url}?{request.query_string.decode('utf-8')}"

    logger.info(f"Proxying {request.method} request to {url}")

    # Prepare headers (exclude hop-by-hop headers)
    headers = {
        key: value for key, value in request.headers.items()
        if key.lower() not in ['host', 'connection', 'transfer-encoding']
    }

    # Add authentication
    auth = HTTPBasicAuth(OPENSEARCH_USERNAME, OPENSEARCH_PASSWORD)

    try:
        # Forward the request to OpenSearch
        response = requests.request(
            method=request.method,
            url=url,
            headers=headers,
            data=request.get_data(),
            auth=auth,
            verify=False,  # Disable SSL verification (dev only)
            allow_redirects=False,
            timeout=30
        )

        # Build response
        excluded_headers = ['connection', 'keep-alive', 'transfer-encoding', 'content-encoding']
        response_headers = [
            (name, value) for name, value in response.headers.items()
            if name.lower() not in excluded_headers
        ]

        logger.info(f"OpenSearch responded with status {response.status_code}")

        return Response(
            response.content,
            status=response.status_code,
            headers=response_headers
        )

    except requests.exceptions.RequestException as e:
        logger.error(f"Error proxying request: {e}")
        return Response(
            f"Proxy error: {str(e)}",
            status=502
        )

@app.route('/health', methods=['GET'])
def health():
    """
    Health check endpoint
    """
    try:
        # Check OpenSearch health
        response = requests.get(
            f"{OPENSEARCH_ENDPOINT}/_cluster/health",
            auth=HTTPBasicAuth(OPENSEARCH_USERNAME, OPENSEARCH_PASSWORD),
            verify=False,
            timeout=5
        )

        if response.status_code == 200:
            return {'status': 'healthy', 'opensearch': 'connected'}, 200
        else:
            return {'status': 'unhealthy', 'opensearch': 'unreachable'}, 503

    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return {'status': 'unhealthy', 'error': str(e)}, 503

if __name__ == '__main__':
    logger.info(f"Starting OAuth2 proxy for OpenSearch at {OPENSEARCH_ENDPOINT}")
    logger.info("Proxy listening on http://0.0.0.0:8080")
    app.run(host='0.0.0.0', port=8080, debug=False)
