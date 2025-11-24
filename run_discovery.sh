#!/bin/bash
set -e

# --- CONFIGURATION ---
BMC_IP="172.24.0.2"
BMC_USER="root"
BMC_PASS="initial0"
CACHE_FILE="/tmp/magellan_assets.db"

# Clear environment proxies
unset http_proxy
unset https_proxy

# Clean slate
rm -f $CACHE_FILE

echo ">>> STEP 1: Configuring Vault..."
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=hms
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault secrets enable -path "secret/hms-creds" -version=1 kv > /dev/null 2>&1 || true
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault kv put secret/hms-creds/x1000c1s7b0 username=$BMC_USER password=$BMC_PASS

echo -e "\n>>> STEP 2: Scanning (Populate Cache)..."
# We scan to populate the local DB.
magellan scan "https://${BMC_IP}:443" \
    --cache "${CACHE_FILE}" \
    --insecure \
    --timeout 30 \
    --log-level debug

echo -e "\n>>> STEP 3: Collecting (Send to SMD)..."
# We collect from the cache. 
# Since we patched the merge bug, this will now succeed and pipe valid JSON to send.
magellan collect \
    --cache "${CACHE_FILE}" \
    --username "${BMC_USER}" \
    --password "${BMC_PASS}" \
    --timeout 30 \
    --log-level debug \
    --show \
    | magellan send http://localhost:27779

echo -e "\n>>> STEP 4: Verifying SMD Data..."
# This should now return the ComponentEndpoints (the Node data)
curl -sS http://localhost:27779/hsm/v2/Inventory/ComponentEndpoints | jq .