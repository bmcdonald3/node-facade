#!/bin/bash
set -e

# --- CONFIGURATION ---
BMC_IP="172.24.0.2"
BMC_USER="root"
BMC_PASS="initial0"
# The XNAME you want to assign to this BMC
BMC_XNAME="x1000c1s7b0" 

echo ">>> STEP 1: Configuring Vault Secrets..."
# We need to make sure the secret engine is on and the credentials are stored
# so PCS can read them later to do power controls.

export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=hms

# 1. Enable the secret engine (KV version 1)
# We use '|| true' so it doesn't fail if already enabled
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault secrets enable -path "secret/hms-creds" -version=1 kv > /dev/null 2>&1 || true

# 2. Write the credentials for the BMC
echo "Writing credentials for $BMC_XNAME..."
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault kv put secret/hms-creds/$BMC_XNAME username=$BMC_USER password=$BMC_PASS

echo -e "\n>>> STEP 2: Running Magellan Discovery..."
# This reaches out to the physical hardware, pulls the Redfish data,
# and pushes it into SMD.

if ! command -v magellan &> /dev/null; then
    echo "Error: 'magellan' command not found."
    echo "Please ensure magellan is in your PATH."
    exit 1
fi

echo "Collecting inventory from $BMC_IP..."
magellan collect "https://${BMC_IP}" \
    --username "${BMC_USER}" \
    --password "${BMC_PASS}" \
    --insecure \
    -v \
    | magellan send http://localhost:27779

echo -e "\n>>> STEP 3: Verifying SMD Data..."

echo "Checking for Redfish Endpoint (BMC):"
curl -sS http://localhost:27779/hsm/v2/Inventory/RedfishEndpoints | jq .

echo "Checking for Component Endpoint (Node):"
# This is the specific table Fabrica needs
curl -sS "http://localhost:27779/hsm/v2/Inventory/ComponentEndpoints" | jq .

echo -e "\n>>> DONE!"
echo "If you see JSON output above, you are ready to run the Fabrica server."