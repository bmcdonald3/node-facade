#!/bin/bash

SMD_URL="http://localhost:27779/hsm/v2/Inventory/RedfishEndpoints"

echo ">>> Deleting RedfishEndpoints..."

IDS=("x1000c1s7b0" "x1000c1s7b1" "x1000c1s7b2" "x1000c0s0b3")

for id in "${IDS[@]}"; do
    echo "Deleting $id..."
    curl -s -o /dev/null -X DELETE "$SMD_URL/$id"
done

curl -s "$SMD_URL" | jq .