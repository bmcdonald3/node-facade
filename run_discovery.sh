#!/bin/bash
set -e

BMC_IP="172.24.0.2"
BMC_USER="root"
BMC_PASS="initial0"

# Clear proxies
unset http_proxy
unset https_proxy

echo ">>> STEP 1: Configuring Vault..."
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=hms
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault secrets enable -path "secret/hms-creds" -version=1 kv > /dev/null 2>&1 || true
docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault kv put secret/hms-creds/x1000c1s7b0 username=$BMC_USER password=$BMC_PASS

echo -e "\n>>> STEP 2: Running Direct Crawl..."

magellan crawl "https://${BMC_IP}:443" \
    --username "${BMC_USER}" \
    --password "${BMC_PASS}" \
    --insecure \
    --timeout 30 \
    --log-level debug \
    --show \
    | magellan send http://localhost:27779

echo -e "\n>>> STEP 3: Verifying SMD Data..."
curl -sS http://localhost:27779/hsm/v2/Inventory/ComponentEndpoints | jq .