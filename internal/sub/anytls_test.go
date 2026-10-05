package sub

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/slinxlink/node/internal/database"
)

func TestAnyTLSShareAfterSwitchingFromTLS2Reality(t *testing.T) {
	ib := database.Inbound{
		Protocol:          "anytls",
		Name:              "AnyTLS Reality",
		Port:              443,
		TLSType:           "reality",
		ServerName:        "old.example.com",
		ALPN:              "h2,http/1.1",
		CipherSuites:      "TLS_AES_128_GCM_SHA256",
		TLSMinVersion:     "1.2",
		TLSMaxVersion:     "1.3",
		Insecure:          true,
		ECHEnabled:        true,
		ECHConfig:         "old-ech-config",
		UTLS:              "safari",
		RealityServerName: "new.example.com",
		RealityPublicKey:  "test-public-key",
		RealityShortIDs:   `["1234abcd"]`,
	}

	link := anytls("password", "proxy.example.com", ib)
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("security") != "reality" || query.Get("sni") != ib.RealityServerName || query.Get("pbk") != ib.RealityPublicKey || query.Get("sid") != "1234abcd" || query.Get("fp") != "safari" {
		t.Errorf("Reality share link is missing client authentication: %s", link)
	}
	if query.Has("alpn") || query.Has("ech") || query.Has("insecure") {
		t.Errorf("old TLS settings leaked into Reality link: %s", link)
	}

	var outbound map[string]any
	if err := json.Unmarshal([]byte(anytlsSingBox("password", "proxy.example.com", ib)), &outbound); err != nil {
		t.Fatal(err)
	}
	tls, ok := outbound["tls"].(map[string]any)
	if !ok || tls["server_name"] != ib.RealityServerName {
		t.Errorf("Reality client SNI is wrong: %#v", outbound)
	}
	reality, ok := tls["reality"].(map[string]any)
	if !ok || reality["enabled"] != true || reality["public_key"] != ib.RealityPublicKey || reality["short_id"] != "1234abcd" {
		t.Errorf("Reality client authentication is missing: %#v", outbound)
	}
	if _, ok := tls["alpn"]; ok {
		t.Errorf("old TLS ALPN leaked into Reality outbound: %#v", tls)
	}
	if _, ok := tls["ech"]; ok {
		t.Errorf("old TLS ECH leaked into Reality outbound: %#v", tls)
	}
	for _, field := range []string{"insecure", "cipher_suites", "min_version", "max_version"} {
		if _, ok := tls[field]; ok {
			t.Errorf("old TLS %s leaked into Reality outbound: %#v", field, tls)
		}
	}
}

func TestAnyTLSTLSShareStillUsesCertificateSettings(t *testing.T) {
	ib := database.Inbound{
		Protocol:          "anytls",
		Name:              "AnyTLS TLS",
		Port:              443,
		TLSType:           "tls",
		ServerName:        "tls.example.com",
		ALPN:              "h2",
		RealityServerName: "unused.example.com",
		RealityPublicKey:  "unused-public-key",
	}
	link := anytls("password", "proxy.example.com", ib)
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("sni") != ib.ServerName || query.Get("alpn") != "h2" || query.Has("pbk") || query.Has("sid") {
		t.Fatalf("TLS link changed unexpectedly: %s", link)
	}
	var outbound map[string]any
	if err := json.Unmarshal([]byte(anytlsSingBox("password", "proxy.example.com", ib)), &outbound); err != nil {
		t.Fatal(err)
	}
	tls := outbound["tls"].(map[string]any)
	if tls["server_name"] != ib.ServerName || tls["reality"] != nil {
		t.Fatalf("TLS outbound changed unexpectedly: %#v", tls)
	}
}
