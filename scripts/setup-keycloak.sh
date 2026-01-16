#!/bin/bash
set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8081}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"
REALM_NAME="${REALM_NAME:-opencode}"
CLIENT_ID="${CLIENT_ID:-opencode-app}"
CLIENT_SECRET="${CLIENT_SECRET:-opencode-secret}"
REDIRECT_URI="${REDIRECT_URI:-http://localhost:5173/auth/callback}"

echo "Configuring Keycloak..."

echo "Waiting for Keycloak to be ready..."
until curl -sf "${KEYCLOAK_URL}/realms/master" > /dev/null 2>&1; do
  echo "Waiting for Keycloak..."
  sleep 5
done

echo "Keycloak is ready!"

echo "Configuring realm and client via Docker exec..."

docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh config credentials \
  --server http://localhost:8080 \
  --realm master \
  --user ${ADMIN_USER} \
  --password ${ADMIN_PASSWORD}

echo "Creating realm: ${REALM_NAME}"
docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh create realms \
  -s realm=${REALM_NAME} \
  -s enabled=true \
  || echo "Realm already exists, skipping..."

echo "Creating client: ${CLIENT_ID}"
docker exec opencode-keycloak /opt/keycloak/bin/kcadm.sh create clients \
  -r ${REALM_NAME} \
  -s clientId=${CLIENT_ID} \
  -s enabled=true \
  -s clientAuthenticatorType=client-secret \
  -s secret=${CLIENT_SECRET} \
  -s publicClient=false \
  -s directAccessGrantsEnabled=true \
  -s standardFlowEnabled=true \
  -s 'redirectUris=["'${REDIRECT_URI}'","http://localhost:5173/*"]' \
  -s 'webOrigins=["http://localhost:5173"]' \
  || echo "Client already exists, skipping..."

echo ""
echo "Configuration complete!"
echo "==========================================="
echo "Keycloak URL: ${KEYCLOAK_URL}"
echo "Admin credentials: ${ADMIN_USER} / ${ADMIN_PASSWORD}"
echo "Realm: ${REALM_NAME}"
echo "Client ID: ${CLIENT_ID}"
echo "Client Secret: ${CLIENT_SECRET}"
echo "Redirect URI: ${REDIRECT_URI}"
echo "==========================================="
echo ""
echo "OIDC Issuer: ${KEYCLOAK_URL}/realms/${REALM_NAME}"
