// Copyright © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT
// This file contains user-customizable reconciliation logic for Node.
//
// ⚠️ This file is safe to edit - it will NOT be overwritten by code generation.
package reconcilers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/example/inventory-v3/pkg/resources/node"
)

// reconcileNode contains custom reconciliation logic.
func (r *NodeReconciler) reconcileNode(ctx context.Context, res *node.Node) error {
	r.Logger.Infof("Reconciling Node: %s (xname: %s)", res.Metadata.Name, res.Spec.Xname)

	// 1. Observer Phase: Query External Systems (SMD & PCS)
	
	// A. Fetch Inventory from SMD
	smdInfo, err := r.getSMDInfo(ctx, res.Spec.Xname)
	if err != nil {
		// Log error but continue - we might still be able to get PCS status
		r.Logger.Errorf("Failed to get SMD info: %v", err)
		res.Status.Message = fmt.Sprintf("SMD Error: %v", err)
	} else {
		// Map SMD data to our Status
		res.Status.IPAddress = smdInfo.RedfishEndpointFQDN
		if len(smdInfo.RedfishSystemInfo.EthernetNICInfo) > 0 {
			res.Status.MACAddress = smdInfo.RedfishSystemInfo.EthernetNICInfo[0].MACAddress
		}
	}

	// B. Fetch Power Status from PCS
	pcsStatus, err := r.getPCSStatus(ctx, res.Spec.Xname)
	if err != nil {
		r.Logger.Errorf("Failed to get PCS status: %v", err)
		res.Status.Phase = "PowerStatusError"
		res.Status.Message = fmt.Sprintf("PCS Error: %v", err)
		// Return error to trigger a retry with backoff
		return err
	}
	
	// Update Status with PCS reality
	res.Status.ActualPowerState = pcsStatus
	res.Status.LastSync = time.Now()

	// 2. Decision Phase: Check for Drift
	if strings.ToLower(res.Spec.PowerState) != strings.ToLower(res.Status.ActualPowerState) {
		res.Status.Phase = "Syncing"
		res.Status.Message = fmt.Sprintf("Drift detected: Want %s, Have %s", res.Spec.PowerState, res.Status.ActualPowerState)
		res.Status.Ready = false
		
		r.Logger.Warnf("DRIFT: Node %s needs power transition to %s", res.Spec.Xname, res.Spec.PowerState)
        // We will implement the Write logic here in the next step
	} else {
		res.Status.Phase = "Ready"
		res.Status.Message = "Node is consistent"
		res.Status.Ready = true
	}

    // Return nil to indicate success. 
    // The controller will check again after the configured 'requeue_delay' (default 5s).
	return nil
}

// --- Helper Functions & External API Structs ---

// getSMDInfo calls SMD to get details from the RedfishEndpoint (BMC)
// because that is the only table we have populated right now.
func (r *NodeReconciler) getSMDInfo(ctx context.Context, xname string) (*SMDResponse, error) {
    // 1. Convert Node Xname (x1000c1s7b0n0) to BMC Xname (x1000c1s7b0)
    // We assume the simple case where we just strip the last 2 chars (n0)
    bmcXname := xname
    if len(xname) > 2 && strings.HasSuffix(xname, "n0") {
        bmcXname = xname[:len(xname)-2]
    }

    client := &http.Client{Timeout: 5 * time.Second}
    
    // TARGETING THE TABLE THAT HAS DATA
    url := fmt.Sprintf("http://localhost:27779/hsm/v2/Inventory/RedfishEndpoints/%s", bmcXname)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("SMD returned status: %s", resp.Status)
    }

    // 2. Decode the RedfishEndpoint response
    var smdData SMDRedfishEndpoint
    if err := json.NewDecoder(resp.Body).Decode(&smdData); err != nil {
        return nil, err
    }

    // 3. Convert to our internal generic response format
    return &SMDResponse{
        ID: smdData.ID,
        RedfishEndpointFQDN: smdData.IPAddress, // Using IPAddress from your JSON
    }, nil
}

// --- Updated Structs ---

// The struct that matches YOUR SPECIFIC JSON output
type SMDRedfishEndpoint struct {
    ID        string `json:"ID"`
    FQDN      string `json:"FQDN"`
    IPAddress string `json:"IPAddress"` // We added this field
}

// Our internal common struct
type SMDResponse struct {
    ID                 string
    RedfishEndpointFQDN string
    RedfishSystemInfo   struct {
        EthernetNICInfo []struct {
            MACAddress string
        }
    }
}

// getPCSStatus calls PCS to get power status
func (r *NodeReconciler) getPCSStatus(ctx context.Context, xname string) (string, error) {
    client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:28007/v1/power-status?xname=%s", xname)
	
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PCS returned status: %s", resp.Status)
	}

	var pcsData PCSStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&pcsData); err != nil {
		return "", err
	}

	if len(pcsData.Status) == 0 {
		return "unknown", nil
	}

	return pcsData.Status[0].PowerState, nil
}

// --- JSON Structs for External Services ---

type PCSStatusResponse struct {
	Status []struct {
		Xname      string `json:"xname"`
		PowerState string `json:"powerState"`
	} `json:"status"`
}