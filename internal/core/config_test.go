package core

import (
	"encoding/json"
	"testing"

	"github.com/slinxlink/node/internal/database"
)

func TestBuildTunnelInbound(t *testing.T) {
	got, err := buildInbound(database.Inbound{
		Protocol:      "tunnel",
		Port:          10080,
		TunnelAddress: "origin.example.com",
		TunnelPort:    443,
		TunnelNetwork: "tcp,udp",
	}, nil)
	if err != nil {
		t.Fatalf("buildInbound: %v", err)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if config["type"] != "direct" {
		t.Fatalf("type = %v, want direct", config["type"])
	}
	if config["override_address"] != "origin.example.com" {
		t.Fatalf("override_address = %v", config["override_address"])
	}
	if config["override_port"] != float64(443) {
		t.Fatalf("override_port = %v", config["override_port"])
	}
	if _, exists := config["network"]; exists {
		t.Fatalf("network = %#v, want omitted for TCP+UDP", config["network"])
	}
}

func TestBuildTunnelInboundWithSingleNetwork(t *testing.T) {
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			got, err := buildInbound(database.Inbound{
				Protocol:      "tunnel",
				Port:          10080,
				TunnelAddress: "origin.example.com",
				TunnelPort:    443,
				TunnelNetwork: network,
			}, nil)
			if err != nil {
				t.Fatalf("buildInbound: %v", err)
			}
			data, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var config map[string]any
			if err := json.Unmarshal(data, &config); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if config["network"] != network {
				t.Fatalf("network = %#v, want %q", config["network"], network)
			}
		})
	}
}
