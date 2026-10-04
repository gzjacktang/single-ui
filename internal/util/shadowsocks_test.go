package util

import "testing"

func TestShadowsocks2022Key(t *testing.T) {
	first, err := GenerateShadowsocks2022Key()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateShadowsocks2022Key()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidShadowsocks2022Key(first) || !ValidShadowsocks2022Key(second) {
		t.Fatalf("generated keys are invalid: %q, %q", first, second)
	}
	if first == second {
		t.Fatal("generated keys must be independent")
	}
	for _, invalid := range []string{"", "password", "YWJj", first + "!"} {
		if ValidShadowsocks2022Key(invalid) {
			t.Fatalf("accepted invalid key %q", invalid)
		}
	}
}
