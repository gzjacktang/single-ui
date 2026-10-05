package util

import (
	"errors"
	"os"
	"path/filepath"
)

// EnsureSboxCommand 给旧安装补上新的管理命令，不覆盖已有文件。
func EnsureSboxCommand(commandDir string) error {
	alias := filepath.Join(commandDir, "sbox")
	legacy := filepath.Join(commandDir, "slinx")
	if _, err := os.Lstat(alias); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(legacy); err != nil {
		return err
	}
	return os.Symlink(legacy, alias)
}
