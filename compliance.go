package main

import (
	"fmt"
	"strings"
	"time"
)

var ptExemptionPreset string

type exemptionPreset struct {
	Reason         string
	LegalReference string
}

var portugueseExemptions = map[string]exemptionPreset{
	"e_learning": {
		Reason:         "Serviços de educação e formação profissional por entidades reconhecidas",
		LegalReference: "Código do IVA, Artigo 9.º, n.º 9",
	},
	"gambling": {
		Reason:         "Atividades de jogos e apostas autorizadas",
		LegalReference: "Código do IVA, Artigo 9.º, n.º 31 e DL 66/2015",
	},
	"insurance_financial": {
		Reason:         "Operações de seguro ou serviços financeiros isentos de IVA",
		LegalReference: "Código do IVA, Artigos 9.º, n.ºs 27 e 28",
	},
}

func applyPortugueseExemptionPreset(invoice *Invoice) {
	if ptExemptionPreset == "" {
		return
	}
	preset, ok := portugueseExemptions[ptExemptionPreset]
	if !ok {
		return
	}
	if invoice.ExemptionReason == "" {
		invoice.ExemptionReason = preset.Reason
	}
	if invoice.LegalReference == "" {
		invoice.LegalReference = preset.LegalReference
	}
}

func validateInvoiceCompliance(invoice Invoice) error {
	var missing []string
	if strings.TrimSpace(invoice.Id) == "" {
		missing = append(missing, "número sequencial de fatura (--id)")
	}
	if strings.TrimSpace(invoice.Date) == "" {
		missing = append(missing, "data de emissão (--date)")
	}
	if strings.TrimSpace(invoice.From) == "" {
		missing = append(missing, "nome/morada do fornecedor (--from)")
	}
	if strings.TrimSpace(invoice.To) == "" {
		missing = append(missing, "nome/morada do cliente (--to)")
	}
	if strings.TrimSpace(invoice.SellerVATID) == "" {
		missing = append(missing, "NIF do fornecedor (--seller-vat-id)")
	}
	if strings.TrimSpace(invoice.BuyerVATID) == "" {
		missing = append(missing, "NIF do cliente (--buyer-vat-id)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("campos obrigatórios em falta: %s", strings.Join(missing, ", "))
	}

	if invoice.Tax == 0 && strings.TrimSpace(invoice.ExemptionReason) == "" {
		return fmt.Errorf("IVA é 0; indique --exemption-reason ou --pt-exemption")
	}
	if invoice.Tax == 0 && strings.TrimSpace(invoice.LegalReference) == "" {
		return fmt.Errorf("IVA é 0; indique --legal-reference ou --pt-exemption")
	}

	if _, err := time.Parse("Jan 02, 2006", invoice.Date); err != nil {
		return fmt.Errorf("formato de data inválido: use \"Jan 02, 2006\"")
	}
	return nil
}
