package service

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/gzjacktang/single-ui/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPanelPortRejectsConflicts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "config.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	t.Cleanup(func() { conn.Close() })
	if err := db.AutoMigrate(&database.Inbound{}); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	busy := listener.Addr().(*net.TCPAddr).Port
	if err := ValidatePanelPort(db, busy, busy); err != nil {
		t.Fatalf("unchanged active port rejected: %v", err)
	}
	for _, port := range []int{0, -1, 65536, 22, busy} {
		if err := ValidatePanelPort(db, port, 2053); err == nil {
			t.Errorf("invalid/unavailable port %d accepted", port)
		}
	}
	free, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	available := free.Addr().(*net.TCPAddr).Port
	free.Close()
	if err := ValidatePanelPort(db, available, 2053); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.Inbound{Port: available}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ValidatePanelPort(db, available, 2053); err == nil {
		t.Fatal("inbound port accepted as panel port")
	}
}
