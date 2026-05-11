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
	// Totals section uses a wider label area to avoid overflow with long labels.
	totalsLabelOffset = 330
	totalsValueOffset = 492

	// Fixed widths for each optional column.
	colWidthDate     = 65.0
	colWidthTime     = 60.0
	colWidthCategory = 90.0
	colWidthQty      = 38.0
	colWidthRate     = 60.0
	colWidthAmount   = 60.0

	pageLeft  = 40.0
	pageRight = 555.0
)

// colPositions holds the computed X position for each active column.
type colPositions struct {
	descWidth                                     float64
	dateX, timeX, categoryX, qtyX, rateX, amountX float64
}

// computeColPositions assigns X positions dynamically based on which columns
// are active, so unused columns do not leave dead space.
func computeColPositions(inv Invoice) colPositions {
	used := 0.0
	if inv.ShowDateColumn {
		used += colWidthDate
	}
	if inv.ShowTimeColumn {
		used += colWidthTime
	}
	if inv.ShowCategoryColumn {
		used += colWidthCategory
	}
	if inv.ShowQuantityColumn {
		used += colWidthQty
	}
	if inv.ShowRateColumn {
		used += colWidthRate
	}
	if inv.ShowAmountColumn {
		used += colWidthAmount
	}

	descWidth := pageRight - pageLeft - used
	if descWidth < 100 {
		descWidth = 100
	}

	x := pageLeft + descWidth
	p := colPositions{descWidth: descWidth}
	if inv.ShowDateColumn {
		p.dateX = x
		x += colWidthDate
	}
	if inv.ShowTimeColumn {
		p.timeX = x
		x += colWidthTime
	}
	if inv.ShowCategoryColumn {
		p.categoryX = x
		x += colWidthCategory
	}
	if inv.ShowQuantityColumn {
		p.qtyX = x
		x += colWidthQty
	}
	if inv.ShowRateColumn {
		p.rateX = x
		x += colWidthRate
	}
	if inv.ShowAmountColumn {
		p.amountX = x
	}
	return p
}

// truncateToWidth shortens text with an ellipsis if it exceeds maxWidth points.
func truncateToWidth(pdf *gopdf.GoPdf, text string, maxWidth float64) string {
	w, _ := pdf.MeasureTextWidth(text)
	if w <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		w, _ = pdf.MeasureTextWidth(candidate)
		if w <= maxWidth {
			return candidate
		}
	}
	return "…"
}

const (
	subtotalLabel    = "Subtotal"
	discountLabel    = "Desconto"
	taxLabel         = "IVA"
	withholdingLabel = "Retenção IRS"
	totalLabel       = "Total"
	totalToPayLabel  = "Total a pagar"
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

func writeTitle(pdf *gopdf.GoPdf, title, id, date, atcud string) {
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
	pdf.Br(20)
	if atcud != "" {
		_ = pdf.SetFont("Inter", "", 9)
		pdf.SetTextColor(120, 120, 120)
		_ = pdf.Cell(nil, "ATCUD: "+atcud)
		pdf.Br(28)
	} else {
		pdf.Br(28)
	}
}

func writeDraftWatermark(pdf *gopdf.GoPdf) {
	_ = pdf.SetFont("Inter-Bold", "", 72)
	pdf.SetTextColor(220, 220, 220)
	pdf.SetXY(95, 330)
	_ = pdf.Cell(nil, "RASCUNHO")
	pdf.SetTextColor(0, 0, 0)
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
	if invoice.SellerVATID == "" && invoice.BuyerVATID == "" && invoice.Reference == "" {
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
	if invoice.Reference != "" {
		_ = pdf.Cell(nil, "Referência: "+invoice.Reference)
		pdf.Br(14)
	}
	pdf.Br(20)
}

func writeHeaderRow(pdf *gopdf.GoPdf, invoice Invoice, cols colPositions) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "DESCRIÇÃO")
	if invoice.ShowDateColumn {
		pdf.SetX(cols.dateX)
		_ = pdf.Cell(nil, "DATA")
	}
	if invoice.ShowTimeColumn {
		pdf.SetX(cols.timeX)
		_ = pdf.Cell(nil, "HORA")
	}
	if invoice.ShowCategoryColumn {
		pdf.SetX(cols.categoryX)
		_ = pdf.Cell(nil, "CATEGORIA")
	}
	if invoice.ShowQuantityColumn {
		pdf.SetX(cols.qtyX)
		_ = pdf.Cell(nil, "QTD.")
	}
	if invoice.ShowRateColumn {
		pdf.SetX(cols.rateX)
		_ = pdf.Cell(nil, "PREÇO UN.")
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(cols.amountX)
		_ = pdf.Cell(nil, "MONTANTE")
	}
	pdf.Br(24)
}

// notesMaxWidth is the column width available for notes before the totals section begins.
const notesMaxWidth = 285.0

func wrapText(pdf *gopdf.GoPdf, text string, maxWidth float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		w, _ := pdf.MeasureTextWidth(candidate)
		if w > maxWidth {
			lines = append(lines, current)
			current = word
		} else {
			current = candidate
		}
	}
	return append(lines, current)
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
		for _, wrapped := range wrapText(pdf, line, notesMaxWidth) {
			_ = pdf.Cell(nil, wrapped)
			pdf.Br(15)
		}
	}
	pdf.Br(48)
}

func writeExemptionReason(pdf *gopdf.GoPdf, code, reason, legalReference string) {
	pdf.SetY(690)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "MOTIVO DE ISENÇÃO DE IVA")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9.5)
	pdf.SetTextColor(0, 0, 0)
	line := code
	if reason != "" {
		if line != "" {
			line += " - " + reason
		} else {
			line = reason
		}
	}
	_ = pdf.Cell(nil, line)
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

func writeRow(pdf *gopdf.GoPdf, invoice Invoice, i int, item string, quantity, rate float64, cols colPositions) {
	_ = pdf.SetFont("Inter", "", 11)
	pdf.SetTextColor(0, 0, 0)

	sym := currencySymbol(invoice.Currency)
	amount := strconv.FormatFloat(quantity*rate, 'f', 2, 64)

	_ = pdf.Cell(nil, truncateToWidth(pdf, item, cols.descWidth-5))
	if invoice.ShowDateColumn {
		pdf.SetX(cols.dateX)
		_ = pdf.Cell(nil, getSliceValue(invoice.ItemDates, i))
	}
	if invoice.ShowTimeColumn {
		pdf.SetX(cols.timeX)
		_ = pdf.Cell(nil, getSliceValue(invoice.ItemTimes, i))
	}
	if invoice.ShowCategoryColumn {
		pdf.SetX(cols.categoryX)
		_ = pdf.Cell(nil, truncateToWidth(pdf, getSliceValue(invoice.ItemCategories, i), colWidthCategory-5))
	}
	if invoice.ShowQuantityColumn {
		pdf.SetX(cols.qtyX)
		_ = pdf.Cell(nil, formatQuantity(quantity))
	}
	if invoice.ShowRateColumn {
		pdf.SetX(cols.rateX)
		_ = pdf.Cell(nil, sym+strconv.FormatFloat(rate, 'f', 2, 64))
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(cols.amountX)
		_ = pdf.Cell(nil, sym+amount)
	}
	pdf.Br(24)
}

func writeTotals(pdf *gopdf.GoPdf, invoice Invoice, subtotal, tax, discount, taxRate, withholding, withholdingRate, totalQty float64) {
	pdf.SetY(600)

	if invoice.ShowQuantityColumn {
		writeQuantityTotal(pdf, totalQty)
	}
	writeTotal(pdf, subtotalLabel, subtotal, invoice.Currency, false)
	if tax > 0 {
		writeTotal(pdf, fmt.Sprintf("%s (%.2f%%)", taxLabel, taxRate*100), tax, invoice.Currency, false)
	}
	if withholding > 0 {
		writeTotal(pdf, fmt.Sprintf("%s (%.2f%%)", withholdingLabel, withholdingRate*100), withholding, invoice.Currency, true)
	}
	if discount > 0 {
		writeTotal(pdf, discountLabel, discount, invoice.Currency, false)
	}
	finalLabel := totalLabel
	if withholding > 0 {
		finalLabel = totalToPayLabel
	}
	writeTotal(pdf, finalLabel, subtotal+tax-withholding-discount, invoice.Currency, false)
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

func writeTotal(pdf *gopdf.GoPdf, label string, total float64, currency string, negative bool) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(12)
	pdf.SetX(totalsValueOffset)
	if label == totalLabel || label == totalToPayLabel {
		_ = pdf.SetFont("Inter-Bold", "", 11.5)
	}
	prefix := ""
	if negative {
		prefix = "-"
	}
	_ = pdf.Cell(nil, prefix+currencySymbol(currency)+strconv.FormatFloat(total, 'f', 2, 64))
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
