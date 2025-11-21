
# OpenCHAMI Node Facade API (Prototype)

## Overview
This project is a Facade API designed to simplify node management. It abstracts the complexity SMD and presents a logical view of compute nodes.

This API provides a single resource: `Node`. Admins declare the desired state (e.g., `PowerState: On`), and the system automatically handles the translation and orchestration required to achieve that state via backend services. This is opposed to the previous splitting between component and redfish endpoints that held information that was usuaully irrelevant and disjoint. 

## Architecture

The system is built using Fabrica and follows a Kubernetes-style Reconciliation pattern:

1.  The API
    * Stores the "User Intent" in a lightweight file-based backend.
    * Exposes a REST API for `Node` resources.
2.  The Reconciler
    * Watches for changes in the API.
    * Queries SMD and PCS to to populate the node's observed status.
    * If the desired PowerState differs from the actual PowerState, it issues command transitions to PCS.

## Dependencies
This prototype assumes the following services are running:
* **SMD:** Read-only access for inventory and mapping (Port 27779).
* **PCS:** Read/Write access for power status and transitions (Port 28007).

## API Reference

### `Node` Resource

**Spec (User Inputs):**
```json
{
  "xname": "x1000c1s7b0n0",
  "powerState": "on" 
}
```

**Status (System Outputs):**
```json
{
  "actualPowerState": "on",
  "ipAddress": "172.24.0.3",
  "macAddress": "a4-bf-01-3f-6b-40",
  "phase": "Ready",
  "lastSync": "2025-11-21T10:00:00Z"
}
```

## Usage

**1. Register a Node (or run discovery script)**
```bash
curl -X POST http://localhost:8080/nodes \
  -d '{"spec": {"xname": "x1000c1s7b0n0", "powerState": "off"}}'
TODO add tick marks

**2. Turn a Node On**
```bash
curl -X PATCH http://localhost:8080/nodes/x1000c1s7b0n0 \
  -d '{"spec": {"powerState": "on"}}'
```