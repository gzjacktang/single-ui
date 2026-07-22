package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	realityScanTimeout     = 8 * time.Second
	realityDiscoverTimeout = 4 * time.Second
	realityScanConcurrency = 32
	realityScanMaxTokens   = 32
	realityDiscoverMaxIPs  = 256
	realityScanMaxTotal    = 512
)

var defaultRealityScanTargets = []string{
	"www.cloudflare.com:443",
	"www.microsoft.com:443",
	"www.amazon.com:443",
	"www.samsung.com:443",
	"www.nvidia.com:443",
	"www.amd.com:443",
	"www.intel.com:443",
	"www.sony.com:443",
}

type RealityScanResult struct {
	Target      string   `json:"target"`
	Host        string   `json:"host"`
	IP          string   `json:"ip"`
	Port        int      `json:"port"`
	Feasible    bool     `json:"feasible"`
	TLSVersion  string   `json:"tls_version"`
	ALPN        string   `json:"alpn"`
	Curve       string   `json:"curve"`
	CertValid   bool     `json:"cert_valid"`
	CertSubject string   `json:"cert_subject"`
	CertIssuer  string   `json:"cert_issuer"`
	ServerNames []string `json:"server_names"`
	LatencyMS   int64    `json:"latency_ms"`
	Reason      string   `json:"reason,omitempty"`
}

type realityTarget struct {
	host     string
	port     int
	discover bool
}

func parseRealityTargets(input string) ([]realityTarget, error) {
	parts := strings.Split(input, ",")
	if strings.TrimSpace(input) == "" {
		parts = defaultRealityScanTargets
	}
	if len(parts) > realityScanMaxTokens {
		return nil, fmt.Errorf("一次最多输入 %d 个目标", realityScanMaxTokens)
	}
	targets := make([]realityTarget, 0, len(parts))
	for _, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.Contains(raw, "://") {
			return nil, fmt.Errorf("目标 %q 格式不正确", raw)
		}
		if strings.Contains(raw, "/") {
			ips, err := enumerateRealityCIDR(raw, realityDiscoverMaxIPs)
			if err != nil {
				return nil, fmt.Errorf("CIDR %q 格式不正确: %w", raw, err)
			}
			for _, ip := range ips {
				targets = append(targets, realityTarget{host: ip, port: 443, discover: true})
			}
			continue
		}
		host, portText, err := net.SplitHostPort(raw)
		if err != nil {
			host, portText = raw, "443"
		}
		host = strings.Trim(strings.TrimSpace(host), "[]")
		if host == "" || strings.ContainsAny(host, " /\\") {
			return nil, fmt.Errorf("目标 %q 主机名不正确", raw)
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("目标 %q 端口不正确", raw)
		}
		targets = append(targets, realityTarget{host: host, port: port})
	}
	return targets, nil
}

func incrementIP(ip net.IP) {
	for index := len(ip) - 1; index >= 0; index-- {
		ip[index]++
		if ip[index] != 0 {
			break
		}
	}
}

func enumerateRealityCIDR(value string, limit int) ([]string, error) {
	_, network, err := net.ParseCIDR(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	addresses := make([]string, 0, limit)
	examined := 0
	for ip := network.IP.Mask(network.Mask); network.Contains(ip) && examined < limit; incrementIP(ip) {
		examined++
		if isPublicScanIP(ip.String()) {
			addresses = append(addresses, ip.String())
		}
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("网段中没有允许扫描的公网地址")
	}
	return addresses, nil
}

func isPublicScanIP(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 { // RFC 6598 CGNAT
			return false
		}
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 0 { // IETF protocol assignments
			return false
		}
		if v4[0] == 198 && v4[1] >= 18 && v4[1] <= 19 { // benchmarking range
			return false
		}
	}
	return true
}

func resolvePublicIP(ctx context.Context, host string) (net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicScanIP(ip.String()) {
			return nil, fmt.Errorf("不允许扫描本机或内网地址")
		}
		return ip, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if isPublicScanIP(addr.IP.String()) {
			return addr.IP, nil
		}
	}
	return nil, fmt.Errorf("目标未解析到公网 IP")
}

func tlsVersion(value uint16) string {
	switch value {
	case tls.VersionTLS13:
		return "1.3"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS10:
		return "1.0"
	default:
		return ""
	}
}

func usableCertificateName(certificate *x509.Certificate) string {
	candidates := append([]string{certificate.Subject.CommonName}, certificate.DNSNames...)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && !strings.HasPrefix(candidate, "*.") && net.ParseIP(candidate) == nil {
			return candidate
		}
	}
	return ""
}

func scanRealityTarget(target realityTarget) RealityScanResult {
	result := RealityScanResult{Host: target.host, Port: target.port, Target: net.JoinHostPort(target.host, strconv.Itoa(target.port))}
	timeout := realityScanTimeout
	if target.discover {
		timeout = realityDiscoverTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ip, err := resolvePublicIP(ctx, target.host)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	result.IP = ip.String()
	address := net.JoinHostPort(ip.String(), strconv.Itoa(target.port))
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		result.Reason = "TCP 连接失败: " + err.Error()
		return result
	}
	defer conn.Close()

	serverName := target.host
	if net.ParseIP(serverName) != nil {
		serverName = ""
	}
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true, // 手动校验，以便返回完整扫描结果。
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences:   []tls.CurveID{tls.X25519, tls.X25519MLKEM768},
		MinVersion:         tls.VersionTLS12,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		result.Reason = "TLS 握手失败: " + err.Error()
		return result
	}
	result.LatencyMS = time.Since(start).Milliseconds()
	state := tlsConn.ConnectionState()
	result.TLSVersion = tlsVersion(state.Version)
	result.ALPN = state.NegotiatedProtocol
	result.Curve = state.CurveID.String()
	if len(state.PeerCertificates) == 0 {
		result.Reason = "目标未返回证书"
		return result
	}

	leaf := state.PeerCertificates[0]
	result.CertSubject = leaf.Subject.CommonName
	result.ServerNames = append(result.ServerNames, leaf.DNSNames...)
	if len(leaf.Issuer.Organization) > 0 {
		result.CertIssuer = leaf.Issuer.Organization[0]
	} else {
		result.CertIssuer = leaf.Issuer.CommonName
	}
	verifyName := serverName
	if verifyName == "" {
		verifyName = usableCertificateName(leaf)
		if verifyName != "" {
			result.Host = verifyName
			result.Target = net.JoinHostPort(verifyName, strconv.Itoa(target.port))
		}
	}
	if verifyName == "" {
		result.Reason = "证书中没有可用的 SNI 域名"
		return result
	}
	intermediates := x509.NewCertPool()
	for _, certificate := range state.PeerCertificates[1:] {
		intermediates.AddCert(certificate)
	}
	_, verifyErr := leaf.Verify(x509.VerifyOptions{DNSName: verifyName, Intermediates: intermediates})
	result.CertValid = verifyErr == nil
	result.Feasible = state.Version == tls.VersionTLS13 && state.NegotiatedProtocol == "h2" &&
		(state.CurveID == tls.X25519 || state.CurveID == tls.X25519MLKEM768) && result.CertValid
	if !result.Feasible {
		switch {
		case verifyErr != nil:
			result.Reason = "证书验证失败: " + verifyErr.Error()
		case state.Version != tls.VersionTLS13:
			result.Reason = "未协商 TLS 1.3"
		case state.NegotiatedProtocol != "h2":
			result.Reason = "未协商 HTTP/2"
		default:
			result.Reason = "未使用 X25519 密钥交换"
		}
	}
	return result
}

func ScanRealityTargets(input string) ([]RealityScanResult, error) {
	var tokens []string
	for _, raw := range strings.Split(input, ",") {
		if token := strings.TrimSpace(raw); token != "" {
			tokens = append(tokens, token)
		}
	}
	if len(tokens) == 0 {
		tokens = append(tokens, defaultRealityScanTargets...)
	}
	if len(tokens) > realityScanMaxTokens {
		return nil, fmt.Errorf("一次最多输入 %d 个目标", realityScanMaxTokens)
	}

	var targets []realityTarget
	var invalid []RealityScanResult
	for _, token := range tokens {
		parsed, err := parseRealityTargets(token)
		if err != nil {
			invalid = append(invalid, RealityScanResult{Target: token, Reason: err.Error()})
			continue
		}
		remaining := realityScanMaxTotal - len(targets)
		if remaining <= 0 {
			break
		}
		if len(parsed) > remaining {
			parsed = parsed[:remaining]
		}
		targets = append(targets, parsed...)
	}
	results := make([]RealityScanResult, len(targets))
	sem := make(chan struct{}, realityScanConcurrency)
	var wg sync.WaitGroup
	for index, target := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[index] = scanRealityTarget(target)
		}()
	}
	wg.Wait()

	filtered := make([]RealityScanResult, 0, len(results)+len(invalid))
	for index, result := range results {
		if targets[index].discover && result.TLSVersion == "" {
			continue
		}
		filtered = append(filtered, result)
	}
	filtered = append(filtered, invalid...)
	filtered = deduplicateRealityResults(filtered)
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Feasible != filtered[j].Feasible {
			return filtered[i].Feasible
		}
		left, right := filtered[i].LatencyMS, filtered[j].LatencyMS
		return left > 0 && (right == 0 || left < right)
	})
	return filtered, nil
}

func deduplicateRealityResults(results []RealityScanResult) []RealityScanResult {
	indexes := make(map[string]int, len(results))
	deduplicated := make([]RealityScanResult, 0, len(results))
	for _, result := range results {
		if index, exists := indexes[result.Target]; exists {
			previous := deduplicated[index]
			if (result.Feasible && !previous.Feasible) || (result.Feasible == previous.Feasible && result.LatencyMS > 0 && (previous.LatencyMS == 0 || result.LatencyMS < previous.LatencyMS)) {
				deduplicated[index] = result
			}
			continue
		}
		indexes[result.Target] = len(deduplicated)
		deduplicated = append(deduplicated, result)
	}
	return deduplicated
}
