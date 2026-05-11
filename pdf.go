package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/signintech/gopdf"
)

// ── Page geometry ─────────────────────────────────────────────────────────────

const (
	pageWidth   = 595.0
	pageLeft    = 40.0
	pageRight   = 555.0
	headerBandH = 88.0
)

// ── Colour palette ────────────────────────────────────────────────────────────

const (
	hdrR, hdrG, hdrB   = 15, 23, 42      // #0F172A  header band
	accR, accG, accB   = 59, 130, 246    // #3B82F6  accent blue
	fillR, fillG, fillB = 241, 245, 249  // #F1F5F9  table header / fiscal row
	altR, altG, altB   = 248, 250, 252   // #F8FAFC  alternating row / card bg
	divR, divG, divB   = 226, 232, 240   // #E2E8F0  divider lines
	lblR, lblG, lblB   = 100, 116, 139   // #64748B  slate label text
	bdyR, bdyG, bdyB   = 30, 41, 59     // #1E293B  body text
)

// ── Totals section layout ─────────────────────────────────────────────────────

const (
	totalsLabelOffset = 330
	totalsValueOffset = 492
)

// ── Column widths ─────────────────────────────────────────────────────────────

const (
	colWidthDate     = 65.0
	colWidthTime     = 60.0
	colWidthCategory = 90.0
	colWidthQty      = 38.0
	colWidthRate     = 60.0
	colWidthAmount   = 60.0
)

// ── Label constants ───────────────────────────────────────────────────────────

const (
	subtotalLabel    = "Subtotal"
	discountLabel    = "Desconto"
	taxLabel         = "IVA"
	withholdingLabel = "Retenção IRS"
	totalLabel       = "Total"
	totalToPayLabel  = "Total a pagar"
)

// ── Column layout ─────────────────────────────────────────────────────────────

type colPositions struct {
	descWidth                                        float64
	dateX, timeX, categoryX, qtyX, rateX, amountX float64
}

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

// ── Text helpers ──────────────────────────────────────────────────────────────

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

// notesMaxWidth is the available column width for notes before the totals section.
const notesMaxWidth = 270.0

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

// ── Header band ───────────────────────────────────────────────────────────────

// writeHeader draws the full-width dark header band: company info left,
// invoice title / number / date right.
func writeHeader(pdf *gopdf.GoPdf, invoice Invoice, atcud string) {
	// Dark background band
	pdf.SetFillColor(hdrR, hdrG, hdrB)
	pdf.RectFromUpperLeftWithStyle(0, 0, pageWidth, headerBandH, "F")

	// ── Left: logo + company name ─────────────────────────────────────────────
	logoEndY := 16.0
	if invoice.Logo != "" {
		iw, ih := getImageDimension(invoice.Logo)
		scaledH := 36.0
		scaledW := float64(iw) * scaledH / float64(ih)
		_ = pdf.Image(invoice.Logo, pageLeft, 12, &gopdf.Rect{W: scaledW, H: scaledH})
		logoEndY = 12 + scaledH + 6
	}

	pdf.SetXY(pageLeft, logoEndY)
	fromLines := strings.Split(strings.ReplaceAll(invoice.From, `\n`, "\n"), "\n")
	for i, line := range fromLines {
		pdf.SetX(pageLeft)
		if i == 0 {
			_ = pdf.SetFont("Inter-Bold", "", 10)
			pdf.SetTextColor(255, 255, 255)
		} else {
			_ = pdf.SetFont("Inter", "", 8)
			pdf.SetTextColor(160, 185, 215)
		}
		_ = pdf.Cell(nil, truncateToWidth(pdf, line, 240))
		pdf.Br(12)
	}

	// ── Right: FATURA / number / date / ATCUD ────────────────────────────────
	title := invoice.Title
	if title == "" {
		title = "FATURA"
	}
	_ = pdf.SetFont("Inter-Bold", "", 22)
	pdf.SetTextColor(255, 255, 255)
	titleW, _ := pdf.MeasureTextWidth(title)
	pdf.SetXY(pageRight-titleW, 16)
	_ = pdf.Cell(nil, title)

	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(accR, accG, accB)
	numW, _ := pdf.MeasureTextWidth(invoice.Id)
	pdf.SetXY(pageRight-numW, 44)
	_ = pdf.Cell(nil, invoice.Id)

	_ = pdf.SetFont("Inter", "", 8)
	pdf.SetTextColor(160, 185, 215)
	dateW, _ := pdf.MeasureTextWidth(invoice.Date)
	pdf.SetXY(pageRight-dateW, 58)
	_ = pdf.Cell(nil, invoice.Date)

	if atcud != "" {
		_ = pdf.SetFont("Inter", "", 7)
		pdf.SetTextColor(110, 140, 175)
		atcudStr := "ATCUD: " + atcud
		atcudW, _ := pdf.MeasureTextWidth(atcudStr)
		pdf.SetXY(pageRight-atcudW, 71)
		_ = pdf.Cell(nil, atcudStr)
	}

	pdf.SetXY(pageLeft, headerBandH)
}

// ── Info strip (from / to / fiscal details) ───────────────────────────────────

const (
	infoDivX   = 290.0 // x position of vertical divider
	infoRightX = 307.0 // right column start
)

// writeInfoStrip draws the two-column from/to section and the fiscal details row.
func writeInfoStrip(pdf *gopdf.GoPdf, invoice Invoice) {
	startY := headerBandH + 20.0

	// ── Left: DE ──────────────────────────────────────────────────────────────
	pdf.SetXY(pageLeft, startY)
	_ = pdf.SetFont("Inter", "", 7)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "DE")
	pdf.Br(14)

	fromLines := strings.Split(strings.ReplaceAll(invoice.From, `\n`, "\n"), "\n")
	for i, line := range fromLines {
		pdf.SetX(pageLeft)
		if i == 0 {
			_ = pdf.SetFont("Inter-Bold", "", 10)
			pdf.SetTextColor(bdyR, bdyG, bdyB)
		} else {
			_ = pdf.SetFont("Inter", "", 8.5)
			pdf.SetTextColor(lblR, lblG, lblB)
		}
		_ = pdf.Cell(nil, truncateToWidth(pdf, line, infoDivX-pageLeft-8))
		pdf.Br(13)
	}
	leftEndY := pdf.GetY()

	// ── Right: PARA ───────────────────────────────────────────────────────────
	pdf.SetXY(infoRightX, startY)
	_ = pdf.SetFont("Inter", "", 7)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "PARA")
	pdf.Br(14)

	toLines := strings.Split(strings.ReplaceAll(invoice.To, `\n`, "\n"), "\n")
	for i, line := range toLines {
		pdf.SetX(infoRightX)
		if i == 0 {
			_ = pdf.SetFont("Inter-Bold", "", 10)
			pdf.SetTextColor(bdyR, bdyG, bdyB)
		} else {
			_ = pdf.SetFont("Inter", "", 8.5)
			pdf.SetTextColor(lblR, lblG, lblB)
		}
		_ = pdf.Cell(nil, truncateToWidth(pdf, line, pageRight-infoRightX))
		pdf.Br(13)
	}
	rightEndY := pdf.GetY()

	// Use the taller of the two columns
	stripEndY := leftEndY
	if rightEndY > stripEndY {
		stripEndY = rightEndY
	}
	stripEndY += 6

	// Vertical divider between columns
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.5)
	pdf.Line(infoDivX, startY-2, infoDivX, stripEndY)

	// ── Fiscal details row ────────────────────────────────────────────────────
	nextY := stripEndY + 10
	hasFiscal := invoice.SellerVATID != "" || invoice.BuyerVATID != "" || invoice.Reference != ""
	if hasFiscal {
		const rowH = 22.0
		pdf.SetFillColor(fillR, fillG, fillB)
		pdf.RectFromUpperLeftWithStyle(pageLeft, nextY, pageRight-pageLeft, rowH, "F")

		pdf.SetXY(pageLeft+8, nextY+7)
		_ = pdf.SetFont("Inter", "", 8)
		pdf.SetTextColor(lblR, lblG, lblB)

		var parts []string
		if invoice.SellerVATID != "" {
			parts = append(parts, "NIF Fornecedor: "+invoice.SellerVATID)
		}
		if invoice.BuyerVATID != "" {
			parts = append(parts, "NIF Cliente: "+invoice.BuyerVATID)
		}
		if invoice.Reference != "" {
			parts = append(parts, "Ref.: "+invoice.Reference)
		}
		_ = pdf.Cell(nil, strings.Join(parts, "   ·   "))
		nextY += rowH
	}

	pdf.SetXY(pageLeft, nextY+14)
}

// ── Draft watermark ───────────────────────────────────────────────────────────

func writeDraftWatermark(pdf *gopdf.GoPdf) {
	_ = pdf.SetFont("Inter-Bold", "", 72)
	pdf.SetTextColor(220, 220, 220)
	pdf.SetXY(90, 420)
	_ = pdf.Cell(nil, "RASCUNHO")
	pdf.SetTextColor(bdyR, bdyG, bdyB)
}

// ── Item table ────────────────────────────────────────────────────────────────

func writeHeaderRow(pdf *gopdf.GoPdf, invoice Invoice, cols colPositions) {
	rowY := pdf.GetY()
	const rowH = 24.0

	pdf.SetFillColor(fillR, fillG, fillB)
	pdf.RectFromUpperLeftWithStyle(pageLeft, rowY, pageRight-pageLeft, rowH, "F")

	pdf.SetXY(pageLeft, rowY+8)
	_ = pdf.SetFont("Inter", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
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
	pdf.SetXY(pageLeft, rowY+rowH)
}

func writeRow(pdf *gopdf.GoPdf, invoice Invoice, i int, item string, quantity, rate float64, cols colPositions) {
	rowY := pdf.GetY()
	const rowH = 25.0

	// Alternating row background
	if i%2 == 1 {
		pdf.SetFillColor(altR, altG, altB)
		pdf.RectFromUpperLeftWithStyle(pageLeft, rowY, pageRight-pageLeft, rowH, "F")
	}

	// Bottom divider
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.3)
	pdf.Line(pageLeft, rowY+rowH, pageRight, rowY+rowH)

	sym := currencySymbol(invoice.Currency)
	amount := strconv.FormatFloat(quantity*rate, 'f', 2, 64)

	pdf.SetXY(pageLeft, rowY+8)
	_ = pdf.SetFont("Inter", "", 10)
	pdf.SetTextColor(bdyR, bdyG, bdyB)
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
	pdf.SetXY(pageLeft, rowY+rowH)
}

// ── Notes section ─────────────────────────────────────────────────────────────

func writeNotes(pdf *gopdf.GoPdf, notes string) {
	pdf.SetY(582)
	_ = pdf.SetFont("Inter", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "OBSERVAÇÕES")
	pdf.Br(15)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(bdyR, bdyG, bdyB)

	for _, line := range strings.Split(strings.ReplaceAll(notes, `\n`, "\n"), "\n") {
		for _, wrapped := range wrapText(pdf, line, notesMaxWidth) {
			_ = pdf.Cell(nil, wrapped)
			pdf.Br(14)
		}
	}
}

// ── Exemption reason ──────────────────────────────────────────────────────────

func writeExemptionReason(pdf *gopdf.GoPdf, code, reason, legalReference string) {
	pdf.SetY(695)
	_ = pdf.SetFont("Inter", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "MOTIVO DE ISENÇÃO DE IVA")
	pdf.Br(15)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(bdyR, bdyG, bdyB)
	line := code
	if reason != "" {
		if line != "" {
			line += " - " + reason
		} else {
			line = reason
		}
	}
	_ = pdf.Cell(nil, line)
	pdf.Br(13)
	if legalReference != "" {
		_ = pdf.SetFont("Inter", "", 8)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, "Ref. Legal: "+legalReference)
		pdf.Br(13)
	}
}

// ── Totals card ───────────────────────────────────────────────────────────────

func writeTotals(pdf *gopdf.GoPdf, invoice Invoice, subtotal, tax, discount, taxRate, withholding, withholdingRate, totalQty float64) {
	const baseY = 580.0

	// Count rows to compute card height before drawing
	rows := 1 // subtotal always present
	if invoice.ShowQuantityColumn {
		rows++
	}
	if tax > 0 {
		rows++
	}
	if withholding > 0 {
		rows++
	}
	if discount > 0 {
		rows++
	}
	rows++ // grand total

	const cardX = totalsLabelOffset - 12.0
	const cardW = pageRight - cardX
	cardH := float64(rows)*26 + 28 // +28 for top/bottom padding + separator
	cardY := baseY - 10

	// Card background + border
	pdf.SetFillColor(altR, altG, altB)
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.7)
	pdf.RectFromUpperLeftWithStyle(cardX, cardY, cardW, cardH, "FD")

	pdf.SetY(baseY)

	if invoice.ShowQuantityColumn {
		writeQuantityTotal(pdf, totalQty)
	}
	writeTotalLine(pdf, subtotalLabel, subtotal, invoice.Currency, false, false)
	if tax > 0 {
		writeTotalLine(pdf, fmt.Sprintf("%s (%.2f%%)", taxLabel, taxRate*100), tax, invoice.Currency, false, false)
	}
	if withholding > 0 {
		writeTotalLine(pdf, fmt.Sprintf("%s (%.2f%%)", withholdingLabel, withholdingRate*100), withholding, invoice.Currency, true, false)
	}
	if discount > 0 {
		writeTotalLine(pdf, discountLabel, discount, invoice.Currency, false, false)
	}

	// Thin separator above grand total
	sepY := pdf.GetY() + 3
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.5)
	pdf.Line(cardX+10, sepY, pageRight-10, sepY)
	pdf.Br(10)

	finalLabel := totalLabel
	if withholding > 0 {
		finalLabel = totalToPayLabel
	}
	writeTotalLine(pdf, finalLabel, subtotal+tax-withholding-discount, invoice.Currency, false, true)

	// Position below card for due date / payment terms
	pdf.SetY(cardY + cardH + 8)
}

func writeQuantityTotal(pdf *gopdf.GoPdf, totalQty float64) {
	_ = pdf.SetFont("Inter", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Total Qtd.")
	_ = pdf.SetFont("Inter", "", 10)
	pdf.SetTextColor(bdyR, bdyG, bdyB)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, formatQuantity(totalQty))
	pdf.Br(26)
}

// writeTotalLine renders one row in the totals card.
// negative: prefix value with "−"
// bold: render in accent blue + Inter-Bold (grand total line)
func writeTotalLine(pdf *gopdf.GoPdf, label string, total float64, currency string, negative, bold bool) {
	if bold {
		_ = pdf.SetFont("Inter-Bold", "", 9)
	} else {
		_ = pdf.SetFont("Inter", "", 8)
	}
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, label)

	if bold {
		_ = pdf.SetFont("Inter-Bold", "", 12)
		pdf.SetTextColor(accR, accG, accB)
	} else {
		_ = pdf.SetFont("Inter", "", 10)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
	}
	prefix := ""
	if negative {
		prefix = "-"
	}
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, prefix+currencySymbol(currency)+strconv.FormatFloat(total, 'f', 2, 64))
	pdf.Br(26)
}

func writeDueDate(pdf *gopdf.GoPdf, due string) {
	_ = pdf.SetFont("Inter", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Data de Vencimento")
	_ = pdf.SetFont("Inter", "", 10)
	pdf.SetTextColor(bdyR, bdyG, bdyB)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, due)
	pdf.Br(16)
}

func writePaymentTerms(pdf *gopdf.GoPdf, terms string) {
	_ = pdf.SetFont("Inter", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "Condições de Pagamento")
	_ = pdf.SetFont("Inter", "", 10)
	pdf.SetTextColor(bdyR, bdyG, bdyB)
	pdf.SetX(totalsValueOffset)
	_ = pdf.Cell(nil, terms)
	pdf.Br(16)
}

// ── Footer ────────────────────────────────────────────────────────────────────

func writeFooter(pdf *gopdf.GoPdf, id string) {
	pdf.SetY(812)
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.5)
	pdf.Line(pageLeft, pdf.GetY(), pageRight, pdf.GetY())
	pdf.Br(10)
	_ = pdf.SetFont("Inter", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	idW, _ := pdf.MeasureTextWidth(id)
	pdf.SetX((pageWidth - idW) / 2)
	_ = pdf.Cell(nil, id)
}

// ── Utilities ─────────────────────────────────────────────────────────────────

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
