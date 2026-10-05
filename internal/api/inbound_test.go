package api

import (
	"testing"

	"github.com/slinxlink/node/internal/database"
)

func TestNormalizeInboundSecurityForAnyTLSReality(t *testing.T) {
	ib := database.Inbound{
		Protocol:          "anytls",
		TLSType:           "reality",
		ServerName:        "old.example.com",
		Certs:             "[1]",
		ALPN:              "h2,http/1.1",
		CipherSuites:      "TLS_AES_128_GCM_SHA256",
		TLSMinVersion:     "1.2",
		TLSMaxVersion:     "1.3",
		Insecure:          true,
		ECHEnabled:        true,
		ECHKey:            "old-key",
		ECHConfig:         "old-config",
		UTLS:              "safari",
		RealityServerName: "new.example.com",
		RealityPrivateKey: "new-private-key",
		Flow:              "xtls-rprx-vision",
	}

	normalizeInboundSecurity(&ib)
	if ib.ServerName != "" || ib.Certs != "" || ib.ALPN != "" || ib.CipherSuites != "" || ib.TLSMinVersion != "" || ib.TLSMaxVersion != "" || ib.Insecure || ib.ECHEnabled || ib.ECHKey != "" || ib.ECHConfig != "" {
		t.Fatalf("old TLS configuration was not cleared: %#v", ib)
	}
	if ib.RealityServerName != "new.example.com" || ib.RealityPrivateKey != "new-private-key" || ib.UTLS != "safari" {
		t.Fatalf("Reality configuration was changed: %#v", ib)
	}
	if ib.Flow != "" {
		t.Fatalf("AnyTLS must not retain VLESS Vision flow: %q", ib.Flow)
	}
}

func TestNormalizeInboundSecurityKeepsVLESSTLS(t *testing.T) {
	ib := database.Inbound{
		Protocol:          "vless",
		TLSType:           "tls",
		ServerName:        "tls.example.com",
		Certs:             "[1]",
		ALPN:              "h2",
		Flow:              "xtls-rprx-vision",
		RealityServerName: "unused.example.com",
	}
	normalizeInboundSecurity(&ib)
	if ib.ServerName != "tls.example.com" || ib.Certs != "[1]" || ib.ALPN != "h2" || ib.Flow != "xtls-rprx-vision" {
		t.Fatalf("VLESS TLS settings changed: %#v", ib)
	}
}
