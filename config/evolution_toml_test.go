package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestEvolutionTOMLValid(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := toml.DecodeFile("../evolution.toml", &cfg); err != nil {
		t.Fatalf("DecodeFile(evolution.toml) error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("evolution.toml is invalid: %v", err)
	}
}
