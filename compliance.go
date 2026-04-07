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
		Reason:         "Education services by recognised establishments",
		LegalReference: "Portuguese VAT Code, Article 9(9)",
	},
	"gambling": {
		Reason:         "Authorised gambling activities",
		LegalReference: "Portuguese VAT Code, Article 9(31) and Decree-Law 66/2015",
	},
	"insurance_financial": {
		Reason:         "Insurance or financial services VAT exemption",
		LegalReference: "Portuguese VAT Code, Articles 9(27) and 9(28)",
	},
}

func applyPortugueseExemptionPreset(invoice *Invoice) {
	if strings.ToUpper(invoice.CountryCode) != "PT" || ptExemptionPreset == "" {
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
	if strings.ToUpper(invoice.CountryCode) != "PT" {
		return nil
	}

	missing := make([]string, 0)
	if strings.TrimSpace(invoice.Id) == "" {
		missing = append(missing, "sequential invoice number (--id)")
	}
	if strings.TrimSpace(invoice.Date) == "" {
		missing = append(missing, "issue date (--date)")
	}
	if strings.TrimSpace(invoice.SupplyDate) == "" {
		missing = append(missing, "supply date (--supply-date)")
	}
	if strings.TrimSpace(invoice.From) == "" {
		missing = append(missing, "supplier name/address (--from)")
	}
	if strings.TrimSpace(invoice.To) == "" {
		missing = append(missing, "recipient name/address (--to)")
	}
	if strings.TrimSpace(invoice.SellerVATID) == "" {
		missing = append(missing, "supplier VAT ID (--seller-vat-id)")
	}
	if strings.TrimSpace(invoice.BuyerVATID) == "" {
		missing = append(missing, "recipient VAT ID (--buyer-vat-id)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing PT invoice fields: %s", strings.Join(missing, ", "))
	}

	if invoice.Tax == 0 && strings.TrimSpace(invoice.ExemptionReason) == "" {
		return fmt.Errorf("VAT is 0 for PT invoice; provide --exemption-reason or --pt-exemption")
	}

	if invoice.Tax == 0 && strings.TrimSpace(invoice.LegalReference) == "" {
		return fmt.Errorf("VAT is 0 for PT invoice; provide --legal-reference or --pt-exemption")
	}

	issueDate, err := time.Parse("Jan 02, 2006", invoice.Date)
	if err != nil {
		return fmt.Errorf("invalid --date format: use \"Jan 02, 2006\"")
	}
	supplyDate, err := time.Parse("Jan 02, 2006", invoice.SupplyDate)
	if err != nil {
		return fmt.Errorf("invalid --supply-date format: use \"Jan 02, 2006\"")
	}

	if businessDaysBetween(supplyDate, issueDate) > 5 {
		return fmt.Errorf("PT invoice must be issued within 5 working days of supply date")
	}

	return nil
}

func businessDaysBetween(start, end time.Time) int {
	if end.Before(start) {
		start, end = end, start
	}
	days := 0
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}
