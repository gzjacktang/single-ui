package setup

import (
	"flag"
	"fmt"
	"net"
	"net/mail"
	"os"
	"strings"
	"unicode"

	"github.com/gzjacktang/single-ui/internal/cert"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/task"
	"github.com/gzjacktang/single-ui/internal/util"
)

type Options struct {
	Port       int
	Path       string
	Username   string
	Password   string
	AccessMode string
	Domain     string
	Email      string
}

func (o *Options) Validate() error {
	if o.Port < 1 || o.Port > 65535 || util.ReservedPorts[o.Port] {
		return fmt.Errorf("面板端口无效或与系统保留端口冲突")
	}
	path, err := util.NormalizePanelPath(o.Path)
	if err != nil {
		return err
	}
	o.Path = path
	o.Username = strings.TrimSpace(o.Username)
	if len(o.Username) < 6 {
		return fmt.Errorf("用户名不能少于 6 位")
	}
	for _, r := range o.Username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("用户名只能包含字母、数字和下划线")
		}
	}
	if len(strings.TrimSpace(o.Password)) < 6 {
		return fmt.Errorf("密码不能少于 6 位")
	}
	if o.AccessMode != "ip" && o.AccessMode != "domain" {
		return fmt.Errorf("访问方式只能是 ip 或 domain")
	}
	if o.AccessMode == "domain" {
		o.Domain = strings.ToLower(strings.TrimSpace(o.Domain))
		if !util.ValidateDomain(o.Domain) {
			return fmt.Errorf("面板域名格式不正确")
		}
		if _, err := mail.ParseAddress(strings.TrimSpace(o.Email)); err != nil {
			return fmt.Errorf("ACME 邮箱格式不正确")
		}
	}
	return nil
}

func runTask(id string, action func(*task.Task)) {
	t := task.New(id)
	drained := make(chan struct{})
	go func() {
		for line := range t.Chan() {
			fmt.Println(line)
		}
		close(drained)
	}()
	action(t)
	<-drained
}

func Run(options Options) error {
	if err := options.Validate(); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	if err != nil {
		return fmt.Errorf("面板端口 %d 已被占用: %w", options.Port, err)
	}
	_ = listener.Close()
	if _, err := database.Init(); err != nil {
		return err
	}

	var config database.Config
	if err := database.DB.First(&config).Error; err != nil {
		return err
	}
	if err := database.DB.Model(&config).Updates(map[string]any{
		"port":         options.Port,
		"path":         options.Path,
		"username":     options.Username,
		"password":     options.Password,
		"domain":       "",
		"sub_enable":   false,
		"sub_path":     "",
		"sub_port":     0,
		"board_enable": false,
	}).Error; err != nil {
		return err
	}

	if options.AccessMode == "domain" {
		acme := database.Acme{Provider: "letsencrypt", Email: strings.TrimSpace(options.Email)}
		if err := database.DB.Create(&acme).Error; err != nil {
			return err
		}
		runTask(fmt.Sprintf("setup-acme-%d", acme.ID), func(t *task.Task) { cert.RegisterAcme(&acme, t) })
		if err := database.DB.First(&acme, acme.ID).Error; err != nil || acme.PrivateKey == "" {
			return fmt.Errorf("ACME 账号注册失败")
		}

		panelCert := database.Cert{Domain: options.Domain, Mode: "http", Acme: acme.ID, AutoRenew: true}
		if err := database.DB.Create(&panelCert).Error; err != nil {
			return err
		}
		runTask("setup-cert-"+options.Domain, func(t *task.Task) { cert.ApplyCert(&panelCert, t) })
		if err := database.DB.First(&panelCert, panelCert.ID).Error; err != nil || panelCert.CertPath == "" || panelCert.KeyPath == "" {
			return fmt.Errorf("域名证书申请失败，请确认域名已解析到本机且 80 端口可访问")
		}
		if err := database.DB.Model(&config).Update("domain", options.Domain).Error; err != nil {
			return err
		}
	}
	if database.SQLDB != nil {
		if err := database.SQLDB.Close(); err != nil {
			return err
		}
	}

	scheme, host := "http", "服务器IP"
	if options.AccessMode == "domain" {
		scheme, host = "https", options.Domain
	}
	fmt.Printf("面板地址: %s://%s:%d%s\n用户名: %s\n", scheme, host, options.Port, options.Path, options.Username)
	return nil
}

func Command(args []string) error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	options := Options{}
	flags.IntVar(&options.Port, "port", 0, "panel port")
	flags.StringVar(&options.Path, "path", "", "panel path")
	flags.StringVar(&options.Username, "username", "", "admin username")
	flags.StringVar(&options.Password, "password", "", "admin password")
	flags.StringVar(&options.AccessMode, "access", "ip", "ip or domain")
	flags.StringVar(&options.Domain, "domain", "", "panel domain")
	flags.StringVar(&options.Email, "email", "", "ACME email")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if options.Password == "" {
		options.Password = os.Getenv("SBOX_SETUP_PASSWORD")
		if options.Password == "" {
			options.Password = os.Getenv("SLINX_SETUP_PASSWORD")
		}
	}
	return Run(options)
}

func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化失败:", err)
		os.Exit(1)
	}
}
