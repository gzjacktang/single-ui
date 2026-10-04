package sub

import (
	"encoding/json"
	"net"
	"net/url"
	"strconv"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func shadowsocksPassword(user database.User, inbound database.Inbound) string {
	if !util.ValidShadowsocks2022Key(inbound.ShadowsocksPassword) || !util.ValidShadowsocks2022Key(user.ShadowsocksKey) {
		return ""
	}
	return inbound.ShadowsocksPassword + ":" + user.ShadowsocksKey
}

func shadowsocks(user database.User, host string, inbound database.Inbound) string {
	password := shadowsocksPassword(user, inbound)
	if password == "" {
		return ""
	}
	uri := url.URL{
		Scheme:   "ss",
		User:     url.UserPassword(util.Shadowsocks2022Method, password),
		Host:     net.JoinHostPort(host, strconv.Itoa(inbound.Port)),
		Fragment: inbound.Name,
	}
	return uri.String()
}

func shadowsocksSingBox(user database.User, host string, inbound database.Inbound) string {
	password := shadowsocksPassword(user, inbound)
	if password == "" {
		return ""
	}
	out := map[string]any{
		"type":        "shadowsocks",
		"tag":         "proxy",
		"server":      host,
		"server_port": inbound.Port,
		"method":      util.Shadowsocks2022Method,
		"password":    password,
	}
	data, _ := json.MarshalIndent(out, "        ", "    ")
	return string(data)
}
