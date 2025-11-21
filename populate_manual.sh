#!/bin/bash
set -e

SMD_URL="http://localhost:27779/hsm/v2"

# Define the 3 nodes found in your logs
# Format: XNAME|IP
NODES=(
    "x1000c1s7b0|172.24.0.2"
    "x1000c1s7b1|172.24.0.3"
    "x1000c1s7b2|172.24.0.4"
)

echo ">>> Populating SMD Manually..."

for node in "${NODES[@]}"; do
    IFS="|" read -r XNAME IP <<< "$node"
    NODE_XNAME="${XNAME}n0" # The compute node is a child of the BMC
    MAC="aa:bb:cc:00:00:${IP##*.}" # Fake MAC based on IP

    echo "Processing $XNAME ($IP)..."

    # 1. Create RedfishEndpoint (BMC)
    curl -sS -o /dev/null -X POST -H "Content-Type: application/json" \
      -d '{
        "RedfishEndpoints": [{
            "ID": "'$XNAME'",
            "FQDN": "'$IP'",
            "RediscoverOnUpdate": false,
            "User": "root",
            "Password": "" 
        }]
      }' $SMD_URL/Inventory/RedfishEndpoints

    # 2. Create Hardware (Node)
    curl -sS -o /dev/null -X POST -H "Content-Type: application/json" \
      -d '{
        "ID": "'$NODE_XNAME'",
        "Type": "Node",
        "Class": "River",
        "TypeString": "Node"
      }' $SMD_URL/Inventory/Hardware

    # 3. Create Ethernet Interface (So we see IP/MAC in Fabrica)
    curl -sS -o /dev/null -X POST -H "Content-Type: application/json" \
      -d '{
        "ID": "'$MAC'",
        "Description": "Mgmt",
        "MACAddress": "'$MAC'",
        "IPAddresses": [{"IPAddress": "'$IP'"}],
        "ComponentID": "'$NODE_XNAME'" 
      }' $SMD_URL/Inventory/EthernetInterfaces
done

echo -e "\n>>> Verification: Checking for ComponentEndpoints..."
# This view aggregates the data we just inserted
curl -sS "$SMD_URL/Inventory/ComponentEndpoints" | jq -r '.ComponentEndpoints[] | "\(.ID) - \(.RedfishEndpointFQDN)"'