package api

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/task"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPanelPortSaveAndCertificateTaskStatus(t *testing.T) {
	oldDB := database.DB
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "panel.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	conn, _ := db.DB()
	t.Cleanup(func() { database.DB = oldDB; conn.Close() })
	if err := db.AutoMigrate(&database.Config{}, &database.Inbound{}); err != nil {
		t.Fatal(err)
	}
	config := database.Config{ID: 1, Port: 2053, Path: "/private/admin", Username: "admin_user", Password: "password", SecretKey: "secret"}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.PUT("/config", UpdateConfig)
	router.GET("/task/:id/status", TaskStatus)
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	config.Port = port
	payload, _ := json.Marshal(config)
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/config", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("save port: %s", response.Body)
	}
	var saved database.Config
	db.First(&saved)
	if saved.Port != port || saved.Password != "password" || saved.SecretKey != "secret" {
		t.Fatalf("unexpected config: %+v", saved)
	}

	task := task.New("cert-example")
	task.Log("INFO", "证书已保存")
	task.Done()
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/task/"+task.ID+"/status", nil))
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"done":true`)) {
			t.Fatalf("task completion lost: %s", response.Body)
		}
	}
}
