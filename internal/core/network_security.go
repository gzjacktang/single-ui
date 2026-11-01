package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gzjacktang/single-ui/internal/netpolicy"
	"github.com/sagernet/sing/common/json"
)

func coreCommand(ctx context.Context, binary string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = netpolicy.Environment(os.Environ())
	return cmd
}

func (m *Manager) checkNetworkPolicy() error {
	data, err := m.ReadConfig()
	if err != nil {
		return err
	}
	// Use the core's JSONC parser so custom configs retain comments/trailing commas.
	value, err := json.UnmarshalExtended[any](data)
	if err != nil {
		return fmt.Errorf("读取核心网络安全配置失败: %w", err)
	}
	return checkMultipath(value)
}

func checkMultipath(value any) error {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.EqualFold(key, "tcp_multi_path") && child == true {
				return fmt.Errorf("安全策略禁止 tcp_multi_path=true（MPTCP）；请设为 false 或删除该选项后重试。此缓解不能替代 Linux 内核安全更新")
			}
			if err := checkMultipath(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := checkMultipath(child); err != nil {
				return err
			}
		}
	}
	return nil
}
