// Copyright © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"context"
	"time"
	"github.com/openchami/fabrica/pkg/resource"
)

// Node represents a Node resource
type Node struct {
	resource.Resource
	Spec   NodeSpec   `json:"spec" validate:"required"`
	Status NodeStatus `json:"status,omitempty"`
}

// NodeSpec defines the desired state of Node	
type NodeSpec struct {
	Xname string `json:"xname" validate:"required"`

	// PowerState is the desired power state of the node.
	// Changing this field triggers the Reconciler to call PCS.
	PowerState string `json:"powerState" validate:"required,oneof=on off"`
}

// NodeStatus defines the observed state of Node
type NodeStatus struct {
	Phase      string `json:"phase,omitempty"`
	Message    string `json:"message,omitempty"`
	Ready      bool   `json:"ready"`
	
	// ActualPowerState is the current state reported by PCS.
	ActualPowerState string `json:"actualPowerState"`

	// IPAddress is the management IP found in SMD (RedfishEndpointFQDN).
	IPAddress string `json:"ipAddress,omitempty"`

	// MACAddress is the primary MAC found in SMD (EthernetNICInfo).
	MACAddress string `json:"macAddress,omitempty"`

	// LastSync is the timestamp of the last successful check with SMD/PCS.
	LastSync time.Time `json:"lastSync,omitempty"`
}

// Validate implements custom validation logic for Node
func (r *Node) Validate(ctx context.Context) error {
	// Add custom validation logic here
	// Example:
	// if r.Spec.Name == "forbidden" {
	//     return errors.New("name 'forbidden' is not allowed")
	// }

	return nil
}
// GetKind returns the kind of the resource
func (r *Node) GetKind() string {
	return "Node"
}

// GetName returns the name of the resource
func (r *Node) GetName() string {
	return r.Metadata.Name
}

// GetUID returns the UID of the resource
func (r *Node) GetUID() string {
	return r.Metadata.UID
}

func init() {
	// Register resource type prefix for storage
	resource.RegisterResourcePrefix("Node", "nod")
}
