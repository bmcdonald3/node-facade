# OpenCHAMI Node Facade API (Prototype)

## Goal
To provide a simplified, logical "Node" API for System Administrators that abstracts away the complexity of SMD.

## Architecture Implemented

1.  Accepts high-level user intent (e.g., `PowerState: "on"`).
2.  The Reconciler:
    * Automatically maps logical Node IDs (`n0`) to physical BMC IPs by querying SMD `RedfishEndpoints`.
    * Detects drift between User Intent and Hardware Reality, triggering transitions via PCS.

## Demo: Proof of Concept

### 1. Register a Node
Manually registered a node to represent hardware found in the inventory.

```bash
curl -X POST http://localhost:8081/nodes \
  -H "Content-Type: application/json" \
  -d '{"xname": "x1000c1s7b0n0", "powerState": "off"}'
```

### 2. Verify Translation 
The API successfully translated the Node Xname to a BMC ID and retrieved the IP address from SMD.

```json
{
  "spec": {
    "xname": "x1000c1s7b0n0",
    "powerState": "off"
  },
  "status": {
    "ipAddress": "172.24.0.2",  <-- SUCCESS: Retrieved from Backend SMD
    "actualPowerState": ""
  }
}
```

### 3. Trigger Power Transition
Updated the user intent to on, triggering reconciler.

```bash
curl -X PATCH http://localhost:8081/nodes/nod-bad3f66d \
  -H "Content-Type: application/json" \
  -d '{"powerState": "on"}'
```

**Server Logs:**
```text
[DEBUG] Processing reconciliation for Node/nod-bad3f66d
[INFO] Reconciling Node: (xname: x1000c1s7b0n0)
[WARN] DRIFT: Node x1000c1s7b0n0 needs power transition to on
[DEBUG] Reconciliation successful
```

## What's done
* Created a API that hides backend complexity.
* Linked Fabrica Resources to SMD Inventory (`RedfishEndpoints`).
* The system is talking to real SMD and PCS endpoints.

## Next Steps

1.  **Fix Magellan Discovery:**
    * This manually injects data into SMD using `curl`.
    * Need to get Magellan discovery working on this system again... 

2.  **PCS Integration Hardening:**
    * Ensure PCS returns valid `PowerState`.

3.  **Refine API:**
    * Move from using Fabrica UIDs (`nod-xyz`) to allowing lookups directly by Xname (`x1000...`) for better SysAdmin UX.
    * How is this actually used?