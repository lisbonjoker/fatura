package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func importData(path string, structure *Invoice, flags *pflag.FlagSet) error {
	fileText, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}

	if strings.HasSuffix(path, ".json") {
		err = importJSON(fileText, structure)
	} else if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		err = importYAML(fileText, structure)
	} else {
		return fmt.Errorf("unsupported file type: %s", path)
	}
	if err != nil {
		return err
	}

	// CLI flags override imported values; flag names use hyphens but JSON tags use underscores.
	flags.Visit(func(f *pflag.Flag) {
		if err != nil {
			return
		}
		key := strings.ReplaceAll(f.Name, "-", "_")
		var b []byte
		if f.Value.Type() != "string" {
			b = []byte(fmt.Sprintf(`{"%s":%s}`, key, f.Value))
		} else {
			b = []byte(fmt.Sprintf(`{"%s":"%s"}`, key, f.Value))
		}
		err = importJSON(b, structure)
	})
	return err
}

func importJSON(text []byte, structure *Invoice) error {
	if !json.Valid(text) {
		return fmt.Errorf("invalid JSON")
	}
	if err := json.Unmarshal(text, structure); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	return nil
}

func importYAML(text []byte, structure *Invoice) error {
	if err := yaml.Unmarshal(text, structure); err != nil {
		return fmt.Errorf("malformed YAML: %w", err)
	}
	return nil
}
