package service

import (
	"fmt"
	"net"

	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
	"gorm.io/gorm"
)

func ValidatePanelPort(db *gorm.DB, port, current int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("面板端口必须在 1–65535 之间")
	}
	if port == current {
		return nil
	}
	if util.ReservedPorts[port] {
		return fmt.Errorf("面板端口不能使用系统保留端口")
	}
	var count int64
	if err := db.Model(&database.Inbound{}).Where("port = ?", port).Count(&count).Error; err != nil {
		return fmt.Errorf("检查入站端口失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("端口 %d 已用于入站", port)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("端口 %d 无法监听或已被占用: %w", port, err)
	}
	return listener.Close()
}
