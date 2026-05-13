package main

import (
	"fmt"
	"strings"
)

var exemptionCodeFlag string

type exemptionPreset struct {
	Reason         string
	LegalReference string
}

var atExemptionCodes = map[string]exemptionPreset{
	"M01": {Reason: "Não sujeição", LegalReference: "Artigo 16.º, n.º 6 do CIVA"},
	"M02": {Reason: "Não tributação", LegalReference: "Decreto-Lei n.º 198/90, de 19 de junho, Artigo 6.º"},
	"M04": {Reason: "Isento", LegalReference: "Artigo 13.º do CIVA"},
	"M05": {Reason: "Isento", LegalReference: "Artigo 14.º do CIVA"},
	"M06": {Reason: "Isento", LegalReference: "Artigo 15.º do CIVA"},
	"M07": {Reason: "Isento", LegalReference: "Artigo 9.º do CIVA"},
	"M09": {Reason: "Sem direito à dedução", LegalReference: "Artigo 9.º do CIVA"},
	"M10": {Reason: "Regime de isenção", LegalReference: "Artigo 53.º do CIVA"},
	"M11": {Reason: "Regime especial do tabaco", LegalReference: "Decreto-Lei n.º 346/85, de 23 de agosto"},
	"M12": {Reason: "Margem de lucro — Agências de Viagens", LegalReference: "Decreto-Lei n.º 221/85, de 3 de julho"},
	"M13": {Reason: "Margem de lucro — Bens em segunda mão", LegalReference: "Decreto-Lei n.º 199/96, de 18 de outubro"},
	"M14": {Reason: "Margem de lucro — Objetos de arte", LegalReference: "Decreto-Lei n.º 199/96, de 18 de outubro"},
	"M15": {Reason: "Margem de lucro — Objetos de coleção e antiguidades", LegalReference: "Decreto-Lei n.º 199/96, de 18 de outubro"},
	"M19": {Reason: "Outras isenções", LegalReference: "Código do IVA"},
	"M20": {Reason: "Regime forfetário", LegalReference: "Artigos 59.º a 62.º do CIVA"},
	"M21": {Reason: "Sem direito à dedução", LegalReference: "Artigo 9.º do CIVA"},
	"M25": {Reason: "Mercadorias em consignação", LegalReference: "Artigo 38.º do CIVA"},
	"M26": {Reason: "Isenção com direito de dedução", LegalReference: "Lei n.º 17/2023, de 14 de abril"},
	"M30": {Reason: "Autoliquidação", LegalReference: "Artigo 2.º do CIVA"},
	"M31": {Reason: "Autoliquidação", LegalReference: "Artigo 2.º, n.º 1, alínea j) do CIVA"},
	"M32": {Reason: "Autoliquidação", LegalReference: "Artigo 2.º, n.º 1, alínea l) do CIVA"},
	"M33": {Reason: "Autoliquidação", LegalReference: "Artigo 2.º, n.º 1, alínea m) do CIVA"},
	"M34": {Reason: "Autoliquidação", LegalReference: "Artigo 2.º, n.º 1, alínea n) do CIVA"},
	"M40": {Reason: "Autoliquidação", LegalReference: "Artigo 6.º, n.º 6, alínea a) do CIVA, a contrário"},
	"M41": {Reason: "Autoliquidação", LegalReference: "Artigo 8.º, n.º 3 do RITI"},
	"M42": {Reason: "Autoliquidação", LegalReference: "Decreto-Lei n.º 21/2007, de 29 de janeiro"},
	"M43": {Reason: "Autoliquidação", LegalReference: "Decreto-Lei n.º 362/99, de 16 de setembro"},
	"M99": {Reason: "Não sujeito ou não tributado", LegalReference: "Código do IVA"},
}

func applyExemptionCode(invoice *Invoice) {
	if exemptionCodeFlag == "" {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(exemptionCodeFlag))
	preset, ok := atExemptionCodes[code]
	if !ok {
		return
	}
	invoice.ExemptionCode = code
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

	if invoice.Tax == 0 && invoice.ExemptionCode == "" && strings.TrimSpace(invoice.ExemptionReason) == "" {
		return fmt.Errorf("IVA é 0; indique --exemption <código> (ex: M07) ou --exemption-reason")
	}

	if _, err := normalizeDate(invoice.Date); err != nil {
		return err
	}
	return nil
}
