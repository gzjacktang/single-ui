package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPanelAssetsHandlerSupportsNestedPanelPath(t *testing.T) {
	dist := fstest.MapFS{
		"assets/main.js":  {Data: []byte("const logo = '/assets/logo.webp'")},
		"assets/main.css": {Data: []byte(".icon { background: url(/assets/icon.woff2) }")},
		"favicon.ico":     {Data: []byte("ico")},
	}
	handler := panelAssetsHandler(dist, "/private/admin/panel")

	for _, path := range []string{
		"/private/admin/panel/assets/main.js",
		"/private/admin/panel/assets/main.css",
		"/private/admin/panel/favicon.ico",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, response.Code)
		}
		if path == "/private/admin/panel/assets/main.css" && !strings.Contains(response.Body.String(), "url(/private/admin/panel/assets/icon.woff2)") {
			t.Fatalf("nested CSS asset was not rewritten: %s", response.Body.String())
		}
		if path == "/private/admin/panel/assets/main.js" && !strings.Contains(response.Body.String(), "'/private/admin/panel/assets/logo.webp'") {
			t.Fatalf("nested JS asset was not rewritten: %s", response.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/assets/main.js", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("root asset status = %d, want 404", response.Code)
	}
}

func TestPanelIndexHTMLSupportsRelativeAssets(t *testing.T) {
	html := []byte(`<head><script src="./assets/main.js"></script><link href="./assets/main.css" rel="stylesheet"><link rel="icon" href="./favicon.ico"></head>`)
	got := string(panelIndexHTML(html, "/private/admin/panel"))
	for _, asset := range []string{
		`src="/private/admin/panel/assets/main.js"`,
		`href="/private/admin/panel/assets/main.css"`,
		`href="/private/admin/panel/favicon.ico"`,
	} {
		if !strings.Contains(got, asset) {
			t.Fatalf("index missing %s: %s", asset, got)
		}
	}
}
