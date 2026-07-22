package util

import "testing"

func TestNormalizePanelPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "single segment", input: "/panel", want: "/panel"},
		{name: "nested path", input: "/private/admin/panel/", want: "/private/admin/panel"},
		{name: "trims whitespace", input: "  /private/panel  ", want: "/private/panel"},
		{name: "missing leading slash", input: "panel", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "root", input: "/", wantErr: true},
		{name: "empty segment", input: "/private//panel", wantErr: true},
		{name: "dot segment", input: "/private/../panel", wantErr: true},
		{name: "query", input: "/private/panel?x=1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePanelPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizePanelPath(%q) expected an error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizePanelPath(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizePanelPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
