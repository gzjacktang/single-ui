package core

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCoreCheckDisablesMPTCP(t *testing.T) {
	t.Setenv("GODEBUG", "http2client=0,multipathtcp=1,multipathtcp=3")
	dir := t.TempDir()
	binary := filepath.Join(dir, "sing-box")
	configPath := filepath.Join(dir, "config.json")
	script := "#!/bin/sh\nif [ \"$GODEBUG\" != 'http2client=0,multipathtcp=0' ]; then echo \"unsafe GODEBUG: $GODEBUG\"; exit 1; fi\n"
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	m := &Manager{BinPath: binary, ConfigPath: configPath}
	if err := m.Check(); err != nil {
		t.Fatalf("core did not receive safe environment: %v", err)
	}
}

func TestCoreRejectsExplicitMPTCP(t *testing.T) {
	for _, entry := range []string{
		`{"inbounds":[{"tcp_multi_path":true}]}`,
		`{"outbounds":[{"tcp_multi_path":true}]}`,
		`{"endpoints":[{"tcp_multi_path":true}]}`,
		`{"inbounds":[{"TCP_MULTI_PATH":true}]}`,
		`{"inbounds":[{"tcp_multi_p\u0061th":true}]}`,
		"{ // custom JSONC config\n \"inbounds\": [{\"tcp_multi_path\": true,}],}",
	} {
		t.Run(entry, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, "sing-box")
			configPath := filepath.Join(dir, "config.json")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(entry), 0600); err != nil {
				t.Fatal(err)
			}
			m := &Manager{BinPath: binary, ConfigPath: configPath}
			if err := m.Check(); err == nil || !strings.Contains(err.Error(), "tcp_multi_path") {
				t.Fatalf("explicit MPTCP was not rejected: %v", err)
			}
			if err := m.Start(); err == nil || !strings.Contains(err.Error(), "tcp_multi_path") {
				t.Fatalf("startup did not reject explicit MPTCP: %v", err)
			}
		})
	}
}

func TestCoreAcceptsPlainTCPConfig(t *testing.T) {
	for _, config := range []string{`{}`, `{"inbounds":[{"tcp_multi_path":false}]}`, "{ // comment\n \"inbounds\": [{\"tcp_multi_path\": false,}],}"} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
		if err := (&Manager{ConfigPath: path}).checkNetworkPolicy(); err != nil {
			t.Fatalf("safe custom config rejected: %v", err)
		}
	}
}

func TestCoreChildProcessDisablesMPTCP(t *testing.T) {
	if os.Getenv("SBOX_TEST_CORE_NETWORK_HELPER") == "1" {
		var listener net.ListenConfig
		if listener.MultipathTCP() {
			t.Fatal("child core still defaults to MPTCP")
		}
		return
	}
	t.Setenv("SBOX_TEST_CORE_NETWORK_HELPER", "1")
	t.Setenv("GODEBUG", "multipathtcp=1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := coreCommand(ctx, binary, "-test.run=^TestCoreChildProcessDisablesMPTCP$").CombinedOutput()
	if err != nil {
		t.Fatalf("child process network policy failed: %v\n%s", err, output)
	}
}
