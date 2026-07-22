package util

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var ReservedPorts = map[int]bool{
	21:   true, // FTP 文件传输
	22:   true, // SSH 远程登录
	23:   true, // Telnet 远程登录
	25:   true, // SMTP 邮件发送
	53:   true, // DNS 域名解析
	80:   true, // HTTP
	110:  true, // POP3 邮件接收
	143:  true, // IMAP 邮件接收
	465:  true, // SMTPS 加密邮件发送
	587:  true, // SMTP 邮件提交
	993:  true, // IMAPS 加密邮件接收
	995:  true, // POP3S 加密邮件接收
	2048: true, // sing-box 流量统计接口
	3306: true, // MySQL 数据库
	3389: true, // RDP Windows 远程桌面
	5432: true, // PostgreSQL 数据库
	6379: true, // Redis 缓存数据库
	9090: true, // sing-box Clash API
}

func ValidatePort(port int, usedPorts []int) string {
	if ReservedPorts[port] {
		return "不能使用系统保留端口"
	}
	for _, p := range usedPorts {
		if port == p {
			return "端口已被占用"
		}
	}
	return ""
}

func ValidateDomain(domain string) bool {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}
	if parts[0] == "" {
		return false
	}
	tld := parts[len(parts)-1]
	if tld == "" || len(tld) < 2 {
		return false
	}
	return true
}

var tagPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
var panelPathSegmentPattern = regexp.MustCompile(`^[a-zA-Z0-9._~-]+$`)

func NormalizePanelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" || !strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("路径必须以 '/' 开头且不能为根路径")
	}
	if strings.ContainsAny(value, "?#\\") {
		return "", fmt.Errorf("路径不能包含查询、锚点或反斜杠")
	}
	value = strings.TrimSuffix(value, "/")
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Path != value {
		return "", fmt.Errorf("路径格式不正确")
	}
	for _, segment := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." || !panelPathSegmentPattern.MatchString(segment) {
			return "", fmt.Errorf("路径每一段只能包含字母、数字、点、下划线、横线或波浪号")
		}
	}
	return value, nil
}

func ValidateTag(tag string) string {
	if tag == "" {
		return "请填写标签"
	}
	if !tagPattern.MatchString(tag) {
		return "标签必须以字母开头，只能包含字母、数字、下划线、横杠"
	}
	return ""
}

func ValidateIPv4CIDR(addr string) bool {
	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		return false
	}
	return ip.To4() != nil
}

func ValidateIPv6CIDR(addr string) bool {
	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		return false
	}
	return ip.To4() == nil && ip.To16() != nil
}
