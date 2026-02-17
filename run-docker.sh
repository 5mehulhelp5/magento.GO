#!/bin/bash
# Build and run GoGento Catalog in Docker, connected to Warden DB and Elasticsearch
# Requires: Warden env running (warden env up)
# Usage: ./run-docker.sh

set -e
cd "$(dirname "$0")"

echo "Building gogento-catalog..."
docker compose build

echo "Starting gogento-catalog (network: vk_default)..."
docker compose up -d

echo ""
echo "GoGento Catalog GraphQL running at http://localhost:8080/graphql"
echo "Test: curl -X POST http://localhost:8080/graphql -H 'Content-Type: application/json' -H 'Store: 1' -d '{\"query\":\"query { products { total_count } }\"}'"
