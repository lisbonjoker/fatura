package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ptReplacer normalises Portuguese characters to ASCII for safe directory names.
var ptReplacer = strings.NewReplacer(
	"ã", "a", "â", "a", "á", "a", "à", "a", "ä", "a",
	"ç", "c",
	"é", "e", "ê", "e", "è", "e", "ë", "e",
	"í", "i", "î", "i", "ì", "i", "ï", "i",
	"ó", "o", "ô", "o", "õ", "o", "ò", "o", "ö", "o",
	"ú", "u", "û", "u", "ù", "u", "ü", "u",
	"ñ", "n",
	"Ã", "a", "Â", "a", "Á", "a", "À", "a",
	"Ç", "c",
	"É", "e", "Ê", "e", "È", "e",
	"Í", "i", "Î", "i",
	"Ó", "o", "Ô", "o", "Õ", "o",
	"Ú", "u", "Û", "u",
)

func sanitizeName(s string) string {
	s = ptReplacer.Replace(s)
	s = strings.ToLower(s)
	var b strings.Builder
	prev := rune('-')
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prev = r
		} else if prev != '-' {
			b.WriteRune('-')
			prev = '-'
		}
	}
	result := strings.Trim(b.String(), "-")
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

// invoicePDFPath returns ~/.invoice/history/<client>/<year>/<month>/<id>-<client>.pdf
func invoicePDFPath(inv Invoice) (string, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return "", err
	}

	clientRaw := inv.To
	if idx := strings.Index(clientRaw, "\n"); idx != -1 {
		clientRaw = clientRaw[:idx]
	}
	client := sanitizeName(strings.TrimSpace(clientRaw))
	if client == "" {
		client = "sem-cliente"
	}

	t, err := time.Parse("Jan 02, 2006", inv.Date)
	if err != nil {
		t = time.Now()
	}

	pdfDir := filepath.Join(dir, "history", client,
		fmt.Sprintf("%d", t.Year()),
		fmt.Sprintf("%02d", int(t.Month())))
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return "", err
	}

	filename := inv.Id + "-" + client + ".pdf"
	return filepath.Join(pdfDir, filename), nil
}

type InvoiceRecord struct {
	Id       string  `json:"id"`
	To       string  `json:"to"`
	Date     string  `json:"date"`
	Total    float64 `json:"total"`
	Currency string  `json:"currency"`
	PDF      string  `json:"pdf"`
	Draft    bool    `json:"draft,omitempty"`
	IssuedAt string  `json:"issued_at"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type GlobalConfig struct {
	SMTP SMTPConfig `yaml:"smtp"`
}

func invoiceConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível encontrar o diretório home: %w", err)
	}
	dir := filepath.Join(home, ".invoice")
	return dir, os.MkdirAll(dir, 0755)
}

func loadGlobalConfig() (GlobalConfig, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return GlobalConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if os.IsNotExist(err) {
		return GlobalConfig{}, nil
	}
	if err != nil {
		return GlobalConfig{}, err
	}
	var cfg GlobalConfig
	return cfg, yaml.Unmarshal(data, &cfg)
}

// nextInvoiceNumber increments the per-year counter and returns INV-YYYY-NNN.
// Draft invoices do not consume a number.
func nextInvoiceNumber() (string, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	counterFile := filepath.Join(dir, "counter.json")

	counters := map[string]int{}
	if data, err := os.ReadFile(counterFile); err == nil {
		_ = json.Unmarshal(data, &counters)
	}

	key := fmt.Sprintf("%d", year)
	counters[key]++
	n := counters[key]

	data, _ := json.MarshalIndent(counters, "", "  ")
	if err := os.WriteFile(counterFile, data, 0644); err != nil {
		return "", fmt.Errorf("erro ao guardar contador: %w", err)
	}
	return fmt.Sprintf("INV-%d-%03d", year, n), nil
}

func saveToHistory(inv Invoice, pdfPath string, total float64, draft bool) error {
	dir, err := invoiceConfigDir()
	if err != nil {
		return err
	}
	histFile := filepath.Join(dir, "history.json")

	var records []InvoiceRecord
	if data, err := os.ReadFile(histFile); err == nil {
		_ = json.Unmarshal(data, &records)
	}

	records = append(records, InvoiceRecord{
		Id:       inv.Id,
		To:       inv.To,
		Date:     inv.Date,
		Total:    total,
		Currency: inv.Currency,
		PDF:      pdfPath,
		Draft:    draft,
		IssuedAt: time.Now().Format(time.RFC3339),
	})

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(histFile, data, 0644)
}

func loadHistory() ([]InvoiceRecord, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "history.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []InvoiceRecord
	return records, json.Unmarshal(data, &records)
}

func saveRecurringTemplate(name string, inv Invoice) error {
	dir, err := invoiceConfigDir()
	if err != nil {
		return err
	}
	recurDir := filepath.Join(dir, "recurring")
	if err := os.MkdirAll(recurDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(inv)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(recurDir, name+".yaml"), data, 0644)
}

func loadRecurringTemplate(name string) (Invoice, error) {
	dir, err := invoiceConfigDir()
	if err != nil {
		return Invoice{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "recurring", name+".yaml"))
	if err != nil {
		return Invoice{}, fmt.Errorf("modelo recorrente %q não encontrado", name)
	}
	var inv Invoice
	return inv, yaml.Unmarshal(data, &inv)
}
