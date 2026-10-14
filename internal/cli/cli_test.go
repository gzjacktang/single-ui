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

func TestPanelPortMenuInputAndCancel(t *testing.T) {
	selected := ""
	m := model{items: []item{{label: "修改面板端口", selectable: true, editPort: true}}, portAction: func(value string) string { selected = value; return "已保存" }}
	press := func(key tea.KeyMsg) { next, _ := m.Update(key); m = next.(model) }
	press(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.editingPort {
		t.Fatal("port editor did not open")
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("23456x")})
	press(tea.KeyMsg{Type: tea.KeyBackspace})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("7")})
	press(tea.KeyMsg{Type: tea.KeyEnter})
	if selected != "23457" || m.editingPort {
		t.Fatalf("port submission = %q, editing = %v", selected, m.editingPort)
	}
	press(tea.KeyMsg{Type: tea.KeyEnter})
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.editingPort || selected != "23457" {
		t.Fatal("cancel changed port")
	}
}
