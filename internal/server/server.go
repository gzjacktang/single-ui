package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/api"
	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

var webFS embed.FS

func Init(fs embed.FS) {
	webFS = fs
}

func StartWeb() error {
	var config database.Config
	database.DB.First(&config)
	panelPath, err := util.NormalizePanelPath(config.Path)
	if err != nil {
		return err
	}

	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	api.RegisterRoutes(r, panelPath)

	dist, _ := fs.Sub(webFS, "web/dist")
	static := panelAssetsHandler(dist, panelPath)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == panelPath {
			c.Redirect(http.StatusTemporaryRedirect, panelPath+"/")
			return
		}

		if strings.HasPrefix(path, panelPath+"/assets/") || path == panelPath+"/favicon.ico" {
			static.ServeHTTP(c.Writer, c.Request)
			return
		}

		if strings.HasPrefix(path, panelPath+"/") && !strings.HasPrefix(path, panelPath+"/api/") {
			data, err := fs.ReadFile(webFS, "web/dist/index.html")
			if err != nil {
				c.Status(500)
				return
			}
			encodedPath, _ := json.Marshal(panelPath)
			html := strings.Replace(
				string(data),
				"</head>",
				fmt.Sprintf(`<script>window.__PANEL_PATH__=%s</script></head>`, encodedPath),
				1,
			)
			html = strings.ReplaceAll(html, `"/assets/`, `"`+panelPath+`/assets/`)
			html = strings.ReplaceAll(html, `"/favicon.ico`, `"`+panelPath+`/favicon.ico`)
			c.Data(200, "text/html; charset=utf-8", []byte(html))
			return
		}

		c.Status(404)
	})

	return runEngine(r, config.Domain, config.Port)
}

func panelAssetsHandler(dist fs.FS, panelPath string) http.Handler {
	static := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if strings.HasPrefix(path, panelPath+"/assets/") || path == panelPath+"/favicon.ico" {
			cloned := request.Clone(request.Context())
			relativePath := strings.TrimPrefix(path, panelPath)
			if strings.HasSuffix(relativePath, ".css") || strings.HasSuffix(relativePath, ".js") {
				name := strings.TrimPrefix(relativePath, "/")
				data, err := fs.ReadFile(dist, name)
				if err == nil {
					content := strings.ReplaceAll(string(data), "/assets/", panelPath+"/assets/")
					contentType := "text/css; charset=utf-8"
					if strings.HasSuffix(relativePath, ".js") {
						contentType = "text/javascript; charset=utf-8"
					}
					data = []byte(content)
					writer.Header().Set("Content-Type", contentType)
					writer.WriteHeader(http.StatusOK)
					_, _ = writer.Write(data)
					return
				}
			}
			cloned.URL.Path = relativePath
			static.ServeHTTP(writer, cloned)
			return
		}
		http.NotFound(writer, request)
	})
}

func StartSub() error {
	var config database.Config
	database.DB.First(&config)

	if !config.SubEnable {
		return nil
	}

	subEngine := gin.New()
	subDist, _ := fs.Sub(webFS, "web/dist")
	subEngine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/assets/") || path == "/favicon.ico" {
			http.FileServer(http.FS(subDist)).ServeHTTP(c.Writer, c.Request)
		}
	})

	subEngine.GET(config.SubPath+"/:token", func(c *gin.Context) {
		accept := c.GetHeader("Accept")
		if strings.Contains(strings.ToLower(accept), "text/html") {
			api.GetSubscriptionPage(c, webFS)
		} else {
			api.GetSubscription(c)
		}
	})

	subEngine.GET(config.SubPath+"/:token/clash", func(c *gin.Context) {
		api.GetClashSubscription(c)
	})

	subEngine.GET(config.SubPath+"/:token/surge", func(c *gin.Context) {
		api.GetSurgeSubscription(c)
	})

	return runEngine(subEngine, config.Domain, config.SubPort)
}

func runEngine(r *gin.Engine, domain string, port int) error {
	addr := fmt.Sprintf(":%d", port)
	if domain != "" {
		var cert database.Cert
		database.DB.Where("domain = ?", domain).First(&cert)
		if cert.CertPath == "" || cert.KeyPath == "" {
			return fmt.Errorf("域名 %s 证书路径未配置", domain)
		}
		go r.RunTLS(addr, cert.CertPath, cert.KeyPath)
	} else {
		go r.Run(addr)
	}
	return nil
}
