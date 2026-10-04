package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func TestBuildShadowsocks2022MultiUserInbound(t *testing.T) {
	serverKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	userKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	users := buildUsers("shadowsocks", []database.User{{Enable: true, Name: "alice", Password: "other-protocol-password", ShadowsocksKey: userKey}}, nil, "")
	got, err := buildInbound(database.Inbound{Protocol: "shadowsocks", Port: 8388, ShadowsocksPassword: serverKey}, users)
	if err != nil {
		t.Fatalf("buildInbound: %v", err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config["type"] != "shadowsocks" || config["method"] != util.Shadowsocks2022Method || config["password"] != serverKey {
		t.Fatalf("unexpected Shadowsocks config: %s", data)
	}
	configUsers, ok := config["users"].([]any)
	if !ok || len(configUsers) != 1 || configUsers[0].(map[string]any)["password"] != userKey {
		t.Fatalf("unexpected Shadowsocks users: %s", data)
	}
	if _, ok := config["tls"]; ok {
		t.Fatalf("Shadowsocks must not inherit TLS settings: %s", data)
	}
}

func TestShadowsocksInboundRequiresActiveUser(t *testing.T) {
	serverKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	users := buildUsers("shadowsocks", []database.User{{Enable: false, Name: "alice", ShadowsocksKey: serverKey}}, nil, "")
	if _, err := buildInbound(database.Inbound{Protocol: "shadowsocks", Port: 8388, ShadowsocksPassword: serverKey}, users); err == nil {
		t.Fatal("disabled last user must not leave a single-user Shadowsocks listener")
	}
}

func TestBuildLegacyShadowsocksMultiUserInbound(t *testing.T) {
	userKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	users := buildUsers("shadowsocks", []database.User{{Enable: true, Name: "alice", ShadowsocksKey: userKey}}, nil, "")
	for _, method := range []string{"aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305"} {
		t.Run(method, func(t *testing.T) {
			got, err := buildInbound(database.Inbound{Protocol: "shadowsocks", Port: 8388, ShadowsocksMethod: method}, users)
			if err != nil {
				t.Fatal(err)
			}
			if got.Method != method || got.Password != "" || len(got.Users) != 1 || got.Users[0].Password != userKey {
				t.Fatalf("unexpected inbound: %#v", got)
			}
		})
	}
	if _, err := buildInbound(database.Inbound{Protocol: "shadowsocks", Port: 8388, ShadowsocksMethod: "chacha20-poly1305"}, users); err == nil {
		t.Fatal("unsupported method must be rejected")
	}
}

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
