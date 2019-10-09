package generator

import "testing"

func TestConfig_Generate(t *testing.T) {
	cfg := Config{
		Name: "xyz",
	}
	script, err := cfg.Generate("/etc/tinc/dnet")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(script.Script))
}
