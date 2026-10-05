package database

import (
	"path/filepath"
	"testing"

	"github.com/gzjacktang/single-ui/internal/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackfillShadowsocksKeysPreservesExistingUsers(t *testing.T) {
	oldDB := DB
	t.Cleanup(func() { DB = oldDB })
	var err error
	DB, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := DB.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}
	user := User{Name: "existing", Token: "token"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := backfillShadowsocksKeys(); err != nil {
		t.Fatal(err)
	}
	if err := DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !util.ValidShadowsocks2022Key(user.ShadowsocksKey) {
		t.Fatalf("invalid migrated key: %q", user.ShadowsocksKey)
	}
	initialKey := user.ShadowsocksKey
	if err := backfillShadowsocksKeys(); err != nil {
		t.Fatal(err)
	}
	if err := DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.ShadowsocksKey != initialKey {
		t.Fatal("existing Shadowsocks key changed on repeated startup")
	}
}
