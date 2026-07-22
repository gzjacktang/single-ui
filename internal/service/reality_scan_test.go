package service

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func TestParseRealityTargets(t *testing.T) {
	targets, err := parseRealityTargets("www.cloudflare.com:443, www.microsoft.com")
	if err != nil {
		t.Fatalf("parseRealityTargets: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(targets))
	}
	if targets[1].host != "www.microsoft.com" || targets[1].port != 443 {
		t.Fatalf("second target = %#v", targets[1])
	}
}

func TestParseRealityTargetsRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{
		"https://www.example.com",
		"www.example.com:0",
		"www.example.com:70000",
		"bad host:443",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseRealityTargets(input); err == nil {
				t.Fatalf("parseRealityTargets(%q) expected an error", input)
			}
		})
	}
}

func TestParseRealityTargetsExpandsPublicCIDR(t *testing.T) {
	targets, err := parseRealityTargets("1.1.1.0/30")
	if err != nil {
		t.Fatalf("parseRealityTargets: %v", err)
	}
	if len(targets) != 4 {
		t.Fatalf("len(targets) = %d, want 4", len(targets))
	}
	if targets[0].host != "1.1.1.0" || !targets[0].discover {
		t.Fatalf("first target = %#v", targets[0])
	}
}

func TestParseRealityTargetsRejectsPrivateCIDR(t *testing.T) {
	if _, err := parseRealityTargets("10.0.0.0/24"); err == nil {
		t.Fatal("expected private CIDR to be rejected")
	}
}

func TestDeduplicateRealityResultsKeepsBestResult(t *testing.T) {
	results := deduplicateRealityResults([]RealityScanResult{
		{Target: "example.com:443", LatencyMS: 90},
		{Target: "example.com:443", Feasible: true, LatencyMS: 120},
		{Target: "example.com:443", Feasible: true, LatencyMS: 70},
	})
	if len(results) != 1 || !results[0].Feasible || results[0].LatencyMS != 70 {
		t.Fatalf("results = %#v", results)
	}
}

func TestScanRealityTargetsKeepsInvalidEntries(t *testing.T) {
	results, err := ScanRealityTargets("bad host:443, https://example.com")
	if err != nil {
		t.Fatalf("ScanRealityTargets: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for _, result := range results {
		if result.Reason == "" {
			t.Fatalf("invalid result has no reason: %#v", result)
		}
	}
}

func TestUsableCertificateName(t *testing.T) {
	certificate := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "*.example.com"},
		DNSNames: []string{"*.example.net", "edge.example.org"},
	}
	if got := usableCertificateName(certificate); got != "edge.example.org" {
		t.Fatalf("usableCertificateName = %q", got)
	}

	certificate.DNSNames = []string{"*.example.net", "1.1.1.1"}
	if got := usableCertificateName(certificate); got != "" {
		t.Fatalf("usableCertificateName = %q, want empty", got)
	}
}

func TestPublicScanIP(t *testing.T) {
	for _, input := range []string{"127.0.0.1", "10.0.0.1", "100.64.0.1", "169.254.1.1", "198.18.0.1", "::1", "fc00::1"} {
		if isPublicScanIP(input) {
			t.Fatalf("isPublicScanIP(%q) = true", input)
		}
	}
	for _, input := range []string{"1.1.1.1", "2606:4700:4700::1111"} {
		if !isPublicScanIP(input) {
			t.Fatalf("isPublicScanIP(%q) = false", input)
		}
	}
}
