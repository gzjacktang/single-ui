package cli

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUninstallMenuRequiresConfirmation(t *testing.T) {
	called := false
	m := model{items: []item{{
		label: "卸载", selectable: true, confirmUninstall: true,
		action: func() string { called = true; return "卸载成功" },
	}}}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if called {
		t.Fatal("卸载菜单按回车后直接执行，未要求确认")
	}
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(model)
	if called {
		t.Fatal("取消确认后仍执行卸载")
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !called {
		t.Fatal("确认后未执行卸载")
	}
}
