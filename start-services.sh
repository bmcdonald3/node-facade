#!/bin/sh

export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=hms
KEYS_PATH="keys"

# Ensure we have the helper functions
# (Assuming bash_functions.sh exists in your dir based on your previous script)
source bash_functions.sh 2>/dev/null || echo "Warning: bash_functions.sh not found, ensuring generate_file works manually if needed"

generate_file() {
    # If your original script relied on bash_functions.sh for this:
    if type gen_access_token >/dev/null 2>&1; then
        gen_access_token > access_token
        get_ca_cert > cacert.pem
    else
        echo "Skipping token generation (helper functions missing), assuming auth disabled or handled elsewhere."
    fi
}

vault_configure_jwt() {
    echo "Configuring Vault JWT..."
    # Check if already configured to avoid errors
    if docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault auth list --format json | jq -e 'has("jwt/")' > /dev/null 2>&1; then
        echo "Vault JWT already configured."
        return
    fi

    docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault auth enable -path=jwt jwt
    docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault write auth/jwt/role/test-role policies="metrics" user_claim="sub" role_type="jwt" bound_audiences="test"
    
    # Create temporary policy file
    cat > policy.yml <<-\EOF
path "secret/hms-creds" {
  capabilities = ["read", "list"]
}
EOF
    docker cp policy.yml vault:/policy.yml
    docker exec -e VAULT_TOKEN=hms vault vault policy write metrics /policy.yml
    
    # Check if keys exist before copying
    if [ -f "$KEYS_PATH/public_key.pem" ]; then
        docker cp $KEYS_PATH/public_key.pem vault:/public_key.pem
        docker exec -e VAULT_TOKEN=hms vault vault write auth/jwt/config jwt_supported_algs=RS256 jwt_validation_pubkeys=@/public_key.pem
    else
        echo "Warning: $KEYS_PATH/public_key.pem not found. Skipping JWT validation pubkey setup."
    fi
}

vault_create_keystore() {
    echo "Creating Vault Keystore..."
    docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault secrets disable secret > /dev/null 2>&1
    docker exec -e VAULT_TOKEN=$VAULT_TOKEN vault vault secrets enable -path "secret/hms-creds" -version=1 kv
}

smd_populate() {
    echo "Populating SMD with test node..."
    curl -sS -X POST -d '{"RedfishEndpoints":[{
      "ID":"x1000c0s0b3",
      "FQDN":"x1000c0s0b3",
      "RediscoverOnUpdate":true,
      "User":"root",
      "Password":"root_password"
    }]}' http://localhost:27779/hsm/v2/Inventory/RedfishEndpoints | jq .
}

# Run the steps
generate_file
vault_configure_jwt
vault_create_keystore
smd_populate