package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/signintech/gopdf"
)

const (
	dateColumnOffset     = 230
	timeColumnOffset     = 295
	categoryColumnOffset = 360
	quantityColumnOffset = 410
	rateColumnOffset     = 450
	amountColumnOffset   = 510

	// Totals section uses a wider label area to avoid overflow with long labels.
	totalsLabelOffset = 330
	totalsValueOffset = 492
)

const (
	subtotalLabel = "Subtotal"
	discountLabel = "Desconto"
	taxLabel      = "IVA"
	totalLabel    = "Total"
)

func writeLogo(pdf *gopdf.GoPdf, logo string, from string) {
	if logo != "" {
		width, height := getImageDimension(logo)
		scaledWidth := 100.0
		scaledHeight := float64(height) * scaledWidth / float64(width)
		_ = pdf.Image(logo, pdf.GetX(), pdf.GetY(), &gopdf.Rect{W: scaledWidth, H: scaledHeight})
		pdf.Br(scaledHeight + 24)
	}
	pdf.SetTextColor(55, 55, 55)

	formattedFrom := strings.ReplaceAll(from, `\n`, "\n")
	fromLines := strings.Split(formattedFrom, "\n")

	for i, line := range fromLines {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 12)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
		}
		_ = pdf.Cell(nil, line)
		if i == 0 {
			pdf.Br(18)
		} else {
			pdf.Br(15)
		}
	}
	pdf.Br(21)
	pdf.SetStrokeColor(225, 225, 225)
	pdf.Line(pdf.GetX(), pdf.GetY(), 260, pdf.GetY())
	pdf.Br(36)
}

func writeTitle(pdf *gopdf.GoPdf, title, id, date string) {
	_ = pdf.SetFont("Inter-Bold", "", 24)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, title)
	pdf.Br(36)
	_ = pdf.SetFont("Inter", "", 12)
	pdf.SetTextColor(100, 100, 100)
	_ = pdf.Cell(nil, "Nº Fatura: ")
	_ = pdf.Cell(nil, id)
	pdf.SetTextColor(150, 150, 150)
	_ = pdf.Cell(nil, "  ·  ")
	pdf.SetTextColor(100, 100, 100)
	_ = pdf.Cell(nil, "Data: ")
	_ = pdf.Cell(nil, date)
	pdf.Br(48)
}

func writeDueDate(pdf *gopdf.GoPdf, due string) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Data de Vencimento")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(11)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, due)
	pdf.Br(12)
}

func writePaymentTerms(pdf *gopdf.GoPdf, terms string) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Condições de Pagamento")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(11)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, terms)
	pdf.Br(12)
}

func writeBillTo(pdf *gopdf.GoPdf, to string) {
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", 9)
	_ = pdf.Cell(nil, "FATURAR A")
	pdf.Br(18)

	formattedTo := strings.ReplaceAll(to, `\n`, "\n")
	toLines := strings.Split(formattedTo, "\n")

	for i, line := range toLines {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 15)
			_ = pdf.Cell(nil, line)
			pdf.Br(20)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
			_ = pdf.Cell(nil, line)
			pdf.Br(15)
		}
	}
	pdf.Br(64)
}

func writeRegulatoryDetails(pdf *gopdf.GoPdf, invoice Invoice) {
	if invoice.SellerVATID == "" && invoice.BuyerVATID == "" {
		return
	}
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", 9)
	_ = pdf.Cell(nil, "DADOS FISCAIS")
	pdf.Br(16)
	_ = pdf.SetFont("Inter", "", 9.5)
	pdf.SetTextColor(55, 55, 55)

	if invoice.SellerVATID != "" {
		_ = pdf.Cell(nil, "NIF Fornecedor: "+invoice.SellerVATID)
		pdf.Br(14)
	}
	if invoice.BuyerVATID != "" {
		_ = pdf.Cell(nil, "NIF Cliente: "+invoice.BuyerVATID)
		pdf.Br(14)
	}
	pdf.Br(20)
}

func writeHeaderRow(pdf *gopdf.GoPdf, invoice Invoice) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "DESCRIÇÃO")
	if invoice.ShowDateColumn {
		pdf.SetX(dateColumnOffset)
		_ = pdf.Cell(nil, "DATA")
	}
	if invoice.ShowTimeColumn {
		pdf.SetX(timeColumnOffset)
		_ = pdf.Cell(nil, "HORA")
	}
	if invoice.ShowCategoryColumn {
		pdf.SetX(categoryColumnOffset)
		_ = pdf.Cell(nil, "CATEGORIA")
	}
	if invoice.ShowQuantityColumn {
		pdf.SetX(quantityColumnOffset)
		_ = pdf.Cell(nil, "QTD.")
	}
	if invoice.ShowRateColumn {
		pdf.SetX(rateColumnOffset)
		_ = pdf.Cell(nil, "PREÇO UN.")
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(amountColumnOffset)
		_ = pdf.Cell(nil, "MONTANTE")
	}
	pdf.Br(24)
}

func writeNotes(pdf *gopdf.GoPdf, notes string) {
	pdf.SetY(600)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "OBSERVAÇÕES")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(0, 0, 0)

	for _, line := range strings.Split(strings.ReplaceAll(notes, `\n`, "\n"), "\n") {
		_ = pdf.Cell(nil, line)
		pdf.Br(15)
	}
	pdf.Br(48)
}

func writeExemptionReason(pdf *gopdf.GoPdf, reason, legalReference string) {
	pdf.SetY(690)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "MOTIVO DE ISENÇÃO DE IVA")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, reason)
	pdf.Br(14)
	if legalReference != "" {
		_ = pdf.SetFont("Inter", "", 8.5)
		pdf.SetTextColor(75, 75, 75)
		_ = pdf.Cell(nil, "Ref. Legal: "+legalReference)
		pdf.Br(14)
	}
}

func writeFooter(pdf *gopdf.GoPdf, id string) {
	pdf.SetY(800)
	_ = pdf.SetFont("Inter", "", 10)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, id)
	pdf.SetStrokeColor(225, 225, 225)
	pdf.Line(pdf.GetX()+10, pdf.GetY()+6, 550, pdf.GetY()+6)
	pdf.Br(48)
}

func writeRow(pdf *gopdf.GoPdf, invoice Invoice, i int, item string, quantity float64, rate float64) {
	_ = pdf.SetFont("Inter", "", 11)
	pdf.SetTextColor(0, 0, 0)

	sym := currencySymbol(invoice.Currency)
	amount := strconv.FormatFloat(quantity*rate, 'f', 2, 64)

	_ = pdf.Cell(nil, item)
	if invoice.ShowDateColumn {
		pdf.SetX(dateColumnOffset)
		_ = pdf.Cell(nil, getSliceValue(invoice.ItemDates, i))
	}
	if invoice.ShowTimeColumn {
		pdf.SetX(timeColumnOffset)
		_ = pdf.Cell(nil, getSliceValue(invoice.ItemTimes, i))
	}
	if invoice.ShowCategoryColumn {
		pdf.SetX(categoryColumnOffset)
		_ = pdf.Cell(nil, getSliceValue(invoice.ItemCategories, i))
	}
	if invoice.ShowQuantityColumn {
		pdf.SetX(quantityColumnOffset)
		_ = pdf.Cell(nil, formatQuantity(quantity))
	}
	if invoice.ShowRateColumn {
		pdf.SetX(rateColumnOffset)
		_ = pdf.Cell(nil, sym+strconv.FormatFloat(rate, 'f', 2, 64))
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(amountColumnOffset)
		_ = pdf.Cell(nil, sym+amount)
	}
	pdf.Br(24)
}

func writeTotals(pdf *gopdf.GoPdf, invoice Invoice, subtotal, tax, discount, taxRate, totalQty float64) {
	pdf.SetY(600)

	if invoice.ShowQuantityColumn {
		writeQuantityTotal(pdf, totalQty)
	}
	writeTotal(pdf, subtotalLabel, subtotal, invoice.Currency)
	if tax > 0 {
		writeTotal(pdf, fmt.Sprintf("%s (%.2f%%)", taxLabel, taxRate*100), tax, invoice.Currency)
	}
	if discount > 0 {
		writeTotal(pdf, discountLabel, discount, invoice.Currency)
	}
	writeTotal(pdf, totalLabel, subtotal+tax-discount, invoice.Currency)
}

func writeQuantityTotal(pdf *gopdf.GoPdf, totalQty float64) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Total Qtd.")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(12)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, formatQuantity(totalQty))
	pdf.Br(24)
}

func writeTotal(pdf *gopdf.GoPdf, label string, total float64, currency string) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(12)
	pdf.SetX(totalsValueOffset)
	if label == totalLabel {
		_ = pdf.SetFont("Inter-Bold", "", 11.5)
	}
	_ = pdf.Cell(nil, currencySymbol(currency)+strconv.FormatFloat(total, 'f', 2, 64))
	pdf.Br(24)
}

func getImageDimension(imagePath string) (int, int) {
	f, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 0, 0
	}
	defer f.Close()

	img, _, err := image.DecodeConfig(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", imagePath, err)
		return 0, 0
	}
	return img.Width, img.Height
}

func getSliceValue(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

func formatQuantity(quantity float64) string {
	return strconv.FormatFloat(quantity, 'f', -1, 64)
}
