package sub

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"

	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
)

func shadowsocksPassword(user database.User, inbound database.Inbound) string {
	method := util.ShadowsocksMethod(inbound.ShadowsocksMethod)
	if !util.SupportedShadowsocksMethod(method) || !util.ValidShadowsocks2022Key(user.ShadowsocksKey) {
		return ""
	}
	if method == util.Shadowsocks2022Method {
		if !util.ValidShadowsocks2022Key(inbound.ShadowsocksPassword) {
			return ""
		}
		return inbound.ShadowsocksPassword + ":" + user.ShadowsocksKey
	}
	return user.ShadowsocksKey
}

func shadowsocks(user database.User, host string, inbound database.Inbound) string {
	password := shadowsocksPassword(user, inbound)
	if password == "" {
		return ""
	}
	method := util.ShadowsocksMethod(inbound.ShadowsocksMethod)
	var userInfo *url.Userinfo
	if method == util.Shadowsocks2022Method {
		userInfo = url.UserPassword(method, password)
	} else {
		userInfo = url.User(base64.RawURLEncoding.EncodeToString([]byte(method + ":" + password)))
	}
	uri := url.URL{
		Scheme:   "ss",
		User:     userInfo,
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
		"method":      util.ShadowsocksMethod(inbound.ShadowsocksMethod),
		"password":    password,
	}
	data, _ := json.MarshalIndent(out, "        ", "    ")
	return string(data)
}
