package xray

import "testing"

func TestTempConfigPathPreservesJSONExtension(t *testing.T) {
	got := tempConfigPath("/usr/local/etc/xray/config.json")
	want := "/usr/local/etc/xray/config.tmp.json"
	if got != want {
		t.Fatalf("tempConfigPath() = %q, want %q", got, want)
	}
}

func TestTempConfigPathAddsJSONWhenMissingExtension(t *testing.T) {
	got := tempConfigPath("/usr/local/etc/xray/config")
	want := "/usr/local/etc/xray/config.tmp.json"
	if got != want {
		t.Fatalf("tempConfigPath() = %q, want %q", got, want)
	}
}
