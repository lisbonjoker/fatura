package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// loadClientConfig reads ~/.invoice/clients/<name>.yaml into an Invoice.
func loadClientConfig(name string) (Invoice, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return Invoice{}, err
	}
	clientsDir := filepath.Join(dir, "clients")
	if err := os.MkdirAll(clientsDir, 0755); err != nil {
		return Invoice{}, err
	}
	clientFile := filepath.Join(clientsDir, name+".yaml")
	data, err := os.ReadFile(clientFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Invoice{}, fmt.Errorf("cliente %q não encontrado em %s", name, clientFile)
		}
		return Invoice{}, err
	}
	var inv Invoice
	if err := yaml.Unmarshal(data, &inv); err != nil {
		return Invoice{}, fmt.Errorf("config de cliente inválida: %w", err)
	}
	return inv, nil
}

// saveClientConfig writes an Invoice as a client template to ~/.invoice/clients/<name>.yaml.
func saveClientConfig(name string, inv Invoice) error {
	dir, err := invoiceConfigDir()
	if err != nil {
		return err
	}
	clientsDir := filepath.Join(dir, "clients")
	if err := os.MkdirAll(clientsDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(inv)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(clientsDir, name+".yaml"), data, 0644)
}
