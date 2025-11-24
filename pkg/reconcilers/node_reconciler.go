// Copyright © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT
// This file contains user-customizable reconciliation logic for Node.
//
// ⚠️ This file is safe to edit - it will NOT be overwritten by code generation.
package reconcilers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		// FIX: Use the flattened field directly
		res.Status.MACAddress = smdInfo.MACAddress
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

		// WRITE LOGIC: Only act if we have a valid target state
		if res.Status.ActualPowerState != "unknown" {
			err := r.sendPCSTransition(ctx, res.Spec.Xname, res.Spec.PowerState)
			if err != nil {
				r.Logger.Errorf("Failed to trigger transition: %v", err)
				res.Status.Message = fmt.Sprintf("Transition failed: %v", err)
			} else {
				res.Status.Message = fmt.Sprintf("Transition to %s started", res.Spec.PowerState)
			}
		}

	} else {
		res.Status.Phase = "Ready"
		res.Status.Message = "Node is consistent"
		res.Status.Ready = true
	}

	// Return nil to indicate success.
	return nil
}

// --- Helper Functions & External API Structs ---

// getSMDInfo calls SMD to get details from the ComponentEndpoint (The Golden Record)
func (r *NodeReconciler) getSMDInfo(ctx context.Context, xname string) (*SMDResponse, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// CORRECT URL: Pointing to the rich view we just populated
	url := fmt.Sprintf("http://localhost:27779/hsm/v2/Inventory/ComponentEndpoints/%s", xname)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SMD returned status: %s", resp.Status)
	}

	var smdData SMDComponentEndpoint
	if err := json.NewDecoder(resp.Body).Decode(&smdData); err != nil {
		return nil, err
	}

	// Map the SMD data to our internal struct
	mac := ""
	if len(smdData.RedfishSystemInfo.EthernetNICInfo) > 0 {
		mac = smdData.RedfishSystemInfo.EthernetNICInfo[0].MACAddress
	}

	return &SMDResponse{
		ID:                  smdData.ID,
		RedfishEndpointFQDN: smdData.RedfishEndpointFQDN,
		MACAddress:          mac,
	}, nil
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

// sendPCSTransition calls PCS to change the power state
func (r *NodeReconciler) sendPCSTransition(ctx context.Context, xname, state string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := "http://localhost:28007/v1/transitions"

	// Capitalize state for PCS (on -> On, off -> Off)
	pcsState := strings.Title(strings.ToLower(state))
	if state == "off" {
		pcsState = "Off" // PCS often prefers "Off" or "Force-Off"
	}

	payload := map[string]interface{}{
		"operation": pcsState,
		"location": []map[string]string{
			{"xname": xname},
		},
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PCS error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// --- JSON Structs for External Services ---

type SMDComponentEndpoint struct {
	ID                  string `json:"ID"`
	RedfishEndpointFQDN string `json:"RedfishEndpointFQDN"` // This is the IP
	RedfishSystemInfo   struct {
		EthernetNICInfo []struct {
			MACAddress string `json:"MACAddress"`
		} `json:"EthernetNICInfo"`
	} `json:"RedfishSystemInfo"`
}

// Internal common struct
type SMDResponse struct {
	ID                  string
	RedfishEndpointFQDN string
	MACAddress          string
}

type PCSStatusResponse struct {
	Status []struct {
		Xname      string `json:"xname"`
		PowerState string `json:"powerState"`
	} `json:"status"`
}