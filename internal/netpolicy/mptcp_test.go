package netpolicy

import (
	"net"
	"os"
	"reflect"
	"testing"
)

func TestEnvironmentDisablesMPTCPAndPreservesOtherSettings(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
		want []string
	}{
		{"empty", nil, []string{"GODEBUG=multipathtcp=0"}},
		{"unrelated variables", []string{"PATH=/bin", "TOKEN=a=b"}, []string{"PATH=/bin", "TOKEN=a=b", "GODEBUG=multipathtcp=0"}},
		{"unrelated debug", []string{"GODEBUG=http2client=0"}, []string{"GODEBUG=http2client=0,multipathtcp=0"}},
		{"enabled", []string{"GODEBUG=multipathtcp=1"}, []string{"GODEBUG=multipathtcp=0"}},
		{"duplicate settings", []string{"GODEBUG=multipathtcp=1,http2server=0,multipathtcp=3"}, []string{"GODEBUG=http2server=0,multipathtcp=0"}},
		{"duplicate variables", []string{"GODEBUG=http2client=0", "PATH=/bin", "GODEBUG=multipathtcp=2,http2server=0"}, []string{"PATH=/bin", "GODEBUG=http2server=0,multipathtcp=0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]string(nil), tc.env...)
			got := Environment(tc.env)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Environment = %v, want %v", got, tc.want)
			}
			if !reflect.DeepEqual(tc.env, original) {
				t.Fatal("input environment was mutated")
			}
			if again := Environment(got); !reflect.DeepEqual(got, again) {
				t.Fatalf("policy is not idempotent: %v", again)
			}
		})
	}
}

func TestEnforceOverridesInheritedGoDebug(t *testing.T) {
	t.Setenv("GODEBUG", "http2client=0,multipathtcp=1")
	if err := Enforce(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GODEBUG"); got != "http2client=0,multipathtcp=0" {
		t.Fatalf("GODEBUG = %q", got)
	}
	var listener net.ListenConfig
	var dialer net.Dialer
	if listener.MultipathTCP() || dialer.MultipathTCP() {
		t.Fatal("Go still enables MPTCP after runtime policy enforcement")
	}
}

func TestBuildDefaultDisablesMPTCP(t *testing.T) {
	t.Setenv("GODEBUG", "")
	var listener net.ListenConfig
	if listener.MultipathTCP() {
		t.Fatal("compiled default must disable MPTCP before main runs")
	}
}
