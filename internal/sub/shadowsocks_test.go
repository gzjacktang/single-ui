package sub

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
)

func TestShadowsocks2022ShareLinkAndSingBoxOutbound(t *testing.T) {
	serverKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	userKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	user := database.User{ShadowsocksKey: userKey}
	inbound := database.Inbound{Protocol: "shadowsocks", Name: "测试 SS", Port: 8388, ShadowsocksPassword: serverKey}

	link := shadowsocks(user, "2001:db8::1", inbound)
	if !strings.Contains(link, "%3A") {
		t.Fatalf("multi-user password delimiter must be percent-encoded: %s", link)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	password, ok := parsed.User.Password()
	if parsed.Scheme != "ss" || parsed.User.Username() != util.Shadowsocks2022Method || !ok || password != serverKey+":"+userKey || parsed.Hostname() != "2001:db8::1" || parsed.Port() != "8388" || parsed.Fragment != inbound.Name {
		t.Fatalf("invalid Shadowsocks share link: %s", link)
	}

	var outbound map[string]any
	if err := json.Unmarshal([]byte(shadowsocksSingBox(user, "example.com", inbound)), &outbound); err != nil {
		t.Fatal(err)
	}
	if outbound["method"] != util.Shadowsocks2022Method || outbound["password"] != serverKey+":"+userKey {
		t.Fatalf("invalid sing-box outbound: %#v", outbound)
	}
}

func TestLegacyShadowsocksShareLinkAndSingBoxOutbound(t *testing.T) {
	userKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	user := database.User{ShadowsocksKey: userKey}
	for _, method := range []string{"aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305"} {
		t.Run(method, func(t *testing.T) {
			inbound := database.Inbound{Protocol: "shadowsocks", Name: "SS", Port: 8388, ShadowsocksMethod: method}
			link := shadowsocks(user, "2001:db8::1", inbound)
			parsed, err := url.Parse(link)
			if err != nil {
				t.Fatal(err)
			}
			encoded := parsed.User.Username()
			decoded, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil || string(decoded) != method+":"+userKey || parsed.Hostname() != "2001:db8::1" {
				t.Fatalf("invalid legacy SS URI: %s", link)
			}
			var outbound map[string]any
			if err := json.Unmarshal([]byte(shadowsocksSingBox(user, "example.com", inbound)), &outbound); err != nil {
				t.Fatal(err)
			}
			if outbound["method"] != method || outbound["password"] != userKey {
				t.Fatalf("invalid outbound: %#v", outbound)
			}
		})
	}
}
