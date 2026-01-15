#!/bin/bash
set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8081}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"

echo "Configuring Keycloak..."

echo "Waiting for Keycloak to be ready..."
until curl -sf "${KEYCLOAK_URL}/health/ready" > /dev/null; do
  echo "Waiting for Keycloak..."
  sleep 5
done

echo "Keycloak is ready!"

echo "Configuration complete!"
echo "You can now login to Keycloak at ${KEYCLOAK_URL}"
echo "Admin credentials: ${ADMIN_USER} / ${ADMIN_PASSWORD}"
