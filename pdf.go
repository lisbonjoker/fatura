package main

// Drop-in replacement for pdf.go — Swiss Grid edition.
// Keeps every exported writer signature identical to the original so main.go
// does not need to change. Visual changes are:
//   • IBM Plex Sans / IBM Plex Mono replace Inter
//   • All labels are mono, uppercase, 8.5pt, slate
//   • Single orange dot is the only chromatic accent (next to the doc title)
//   • Header band is gone; replaced by a 12-col strip + 1-px ink rule
//   • Item table: no row tinting, hairlines only, tabular numerics
//   • Totals: right-aligned mono stack, grand total 26pt Mono-SemiBold
//   • Footer: three-column mono strip with ID · page · seller NIF
//
// New font names main.go must register (see Apply Swiss Grid.html):
//   "Sans"    IBMPlexSans-Regular.ttf
//   "Sans-B"  IBMPlexSans-SemiBold.ttf
//   "Mono"    IBMPlexMono-Regular.ttf
//   "Mono-B"  IBMPlexMono-Medium.ttf

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/signintech/gopdf"
)

// ── Page geometry ─────────────────────────────────────────────────────────────

const (
	pageWidth  = 595.0
	pageLeft   = 40.0
	pageRight  = 555.0
	pageBottom = 802.0

	// In Swiss Grid the header collapses to a thin strip + 1px rule.
	headerStripH = 28.0
	headerBandH  = 56.0 // top of body content
)

// ── Colour palette ────────────────────────────────────────────────────────────

const (
	inkR, inkG, inkB    = 10, 10, 10    // #0A0A0A primary ink
	bdyR, bdyG, bdyB    = 48, 48, 48    // #303030 body text
	lblR, lblG, lblB    = 112, 112, 112 // #707070 mono labels
	divR, divG, divB    = 216, 216, 216 // #D8D8D8 hairlines
	hairR, hairG, hairB = 237, 237, 237 // #EDEDED ultra-light row rule
	fillR, fillG, fillB = 244, 243, 239 // #F4F3EF IBAN / paper tint
	altR, altG, altB    = 255, 255, 255 // no alternating row fill
	accR, accG, accB    = 255, 91, 31   // #FF5B1F orange — used ONLY for the doc-type dot
	hdrR, hdrG, hdrB    = 255, 255, 255 // header background == paper (no band)

	negR, negG, negB = 140, 47, 31 // #8C2F1F minus sign / IRS retention
)

// ── Totals section layout ─────────────────────────────────────────────────────

const (
	totalsLabelOffset = 330.0
	totalsValueOffset = 470.0 // right-aligned values flush to this x
	totalsRightEdge   = pageRight
)

// ── Column widths ─────────────────────────────────────────────────────────────

const (
	colWidthDate     = 65.0
	colWidthTime     = 60.0
	colWidthCategory = 86.0
	colWidthQty      = 36.0
	colWidthRate     = 66.0
	colWidthAmount   = 78.0
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
	descWidth                                     float64
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

// rightAlignedCell measures text and draws it flush to rightX at the current Y.
func rightAlignedCell(pdf *gopdf.GoPdf, text string, rightX float64) {
	w, _ := pdf.MeasureTextWidth(text)
	pdf.SetX(rightX - w)
	_ = pdf.Cell(nil, text)
}

// ── Header strip ──────────────────────────────────────────────────────────────
// 4-column top strip: badge · DOC TYPE + dot · ATCUD · REF.
// Followed by a 1-px ink rule. No dark band.

func writeHeader(pdf *gopdf.GoPdf, invoice Invoice, atcud string) {
	const stripY = 36.0

	// Left: orange accent dot + spaced title
	pdf.SetFillColor(accR, accG, accB)
	pdf.RectFromUpperLeftWithStyle(pageLeft, stripY+3, 8, 8, "F")

	pdf.SetXY(pageLeft+14, stripY)
	_ = pdf.SetFont("Sans-B", "", 11)
	pdf.SetTextColor(inkR, inkG, inkB)
	title := strings.ToUpper(invoice.Title)
	if title == "" {
		title = "FATURA"
	}
	_ = pdf.Cell(nil, expandLetterSpacing(title))

	// ATCUD column + compliance label — only rendered when a code is provided
	if atcud != "" {
		const atcudX = 360.0
		pdf.SetXY(atcudX, stripY)
		_ = pdf.SetFont("Mono", "", 8)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, "ATCUD")
		pdf.SetXY(atcudX, stripY+14)
		_ = pdf.SetFont("Mono-B", "", 10)
		pdf.SetTextColor(inkR, inkG, inkB)
		_ = pdf.Cell(nil, atcud)

		pdf.SetXY(pageLeft+14, stripY+16)
		_ = pdf.SetFont("Sans", "", 8.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, "Documento conforme à AT")
	}

	// Right column: reference (or date if no reference)
	pdf.SetXY(0, stripY)
	_ = pdf.SetFont("Mono", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	rightAlignedCell(pdf, "REFERÊNCIA", pageRight)

	pdf.SetXY(0, stripY+14)
	_ = pdf.SetFont("Mono-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	ref := invoice.Reference
	if ref == "" {
		ref = invoice.Date
	}
	rightAlignedCell(pdf, ref, pageRight)

	// Ink rule under the strip
	ruleY := stripY + 32
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(1)
	pdf.Line(pageLeft, ruleY, pageRight, ruleY)

	pdf.SetXY(pageLeft, ruleY+4)
}

// expandLetterSpacing inserts a hair space between letters for a wide cap effect.
func expandLetterSpacing(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		b.WriteRune(r)
		if i < len(runes)-1 && r != ' ' {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// ── Info strip (DE / PARA / meta row) ────────────────────────────────────────

const (
	infoDivX   = 297.5 // page midline
	infoRightX = 312.0
)

func writeInfoStrip(pdf *gopdf.GoPdf, invoice Invoice) {
	startY := 90.0

	drawParty(pdf, "EMITENTE", invoice.From, invoice.SellerVATID, pageLeft, infoDivX-8, startY)
	rightEnd := drawParty(pdf, "DESTINATÁRIO", invoice.To, invoice.BuyerVATID, infoRightX, pageRight, startY)

	// Use whichever party block ended lower
	leftEnd := pdf.GetY()
	stripEndY := leftEnd
	if rightEnd > stripEndY {
		stripEndY = rightEnd
	}
	stripEndY += 4

	// Vertical mid divider
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.5)
	pdf.Line(infoDivX, startY-4, infoDivX, stripEndY)

	// Hairline below parties
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.5)
	pdf.Line(pageLeft, stripEndY+6, pageRight, stripEndY+6)

	// ── Meta row: emissão · vencimento · condições · moeda ────────────────────
	metaY := stripEndY + 18
	cells := []struct {
		label string
		value string
	}{
		{"DATA EMISSÃO", invoice.Date},
		{"VENCIMENTO", invoice.Due},
		{"CONDIÇÕES", firstNonEmpty(invoice.PaymentTerms, "30 dias")},
		{"MOEDA", strings.ToUpper(invoice.Currency)},
	}
	colW := (pageRight - pageLeft) / float64(len(cells))
	for i, c := range cells {
		x := pageLeft + float64(i)*colW
		pdf.SetXY(x, metaY)
		_ = pdf.SetFont("Mono", "", 7.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, c.label)

		pdf.SetXY(x, metaY+13)
		_ = pdf.SetFont("Mono-B", "", 10)
		pdf.SetTextColor(inkR, inkG, inkB)
		_ = pdf.Cell(nil, truncateToWidth(pdf, c.value, colW-8))
	}

	// Bottom ink rule of the info strip
	bottom := metaY + 32
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(1)
	pdf.Line(pageLeft, bottom, pageRight, bottom)

	pdf.SetXY(pageLeft, bottom+22)
}

func drawParty(pdf *gopdf.GoPdf, label, raw, vatID string, x, rightLimit, startY float64) float64 {
	pdf.SetXY(x, startY)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, label)
	pdf.Br(14)

	lines := strings.Split(strings.ReplaceAll(raw, `\n`, "\n"), "\n")
	for i, line := range lines {
		pdf.SetX(x)
		if i == 0 {
			_ = pdf.SetFont("Sans-B", "", 11)
			pdf.SetTextColor(inkR, inkG, inkB)
		} else {
			_ = pdf.SetFont("Sans", "", 9.5)
			pdf.SetTextColor(bdyR, bdyG, bdyB)
		}
		_ = pdf.Cell(nil, truncateToWidth(pdf, line, rightLimit-x))
		if i == 0 {
			pdf.Br(15)
		} else {
			pdf.Br(13)
		}
	}
	if vatID != "" {
		pdf.SetX(x)
		_ = pdf.SetFont("Mono-B", "", 10)
		pdf.SetTextColor(inkR, inkG, inkB)
		_ = pdf.Cell(nil, vatID)
		pdf.Br(13)
	}
	return pdf.GetY()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// ── Draft watermark ───────────────────────────────────────────────────────────

func writeDraftWatermark(pdf *gopdf.GoPdf) {
	_ = pdf.SetFont("Sans-B", "", 96)
	pdf.SetTextColor(232, 232, 232)
	pdf.SetXY(56, 420)
	_ = pdf.Cell(nil, "RASCUNHO")
	pdf.SetTextColor(bdyR, bdyG, bdyB)
}

// ── Item table ────────────────────────────────────────────────────────────────

func writeHeaderRow(pdf *gopdf.GoPdf, invoice Invoice, cols colPositions) {
	rowY := pdf.GetY()
	const rowH = 22.0

	// Top ink rule
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(pageLeft, rowY, pageRight, rowY)

	pdf.SetXY(pageLeft, rowY+8)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "DESCRIÇÃO")
	if invoice.ShowDateColumn {
		labelInCol(pdf, "DATA", cols.dateX, colWidthDate)
	}
	if invoice.ShowTimeColumn {
		labelInCol(pdf, "HORA", cols.timeX, colWidthTime)
	}
	if invoice.ShowCategoryColumn {
		labelInCol(pdf, "CATEGORIA", cols.categoryX, colWidthCategory)
	}
	if invoice.ShowQuantityColumn {
		labelInCol(pdf, "QTD.", cols.qtyX, colWidthQty)
	}
	if invoice.ShowRateColumn {
		labelInCol(pdf, "PREÇO UN.", cols.rateX, colWidthRate)
	}
	if invoice.ShowAmountColumn {
		labelInCol(pdf, "MONTANTE", cols.amountX, colWidthAmount)
	}

	// Bottom hairline
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.4)
	pdf.Line(pageLeft, rowY+rowH, pageRight, rowY+rowH)

	pdf.SetXY(pageLeft, rowY+rowH)
}

// labelInCol right-aligns a column header within its cell, leaving a small
// right pad so it lines up with the values below.
func labelInCol(pdf *gopdf.GoPdf, text string, colX, colW float64) {
	w, _ := pdf.MeasureTextWidth(text)
	pdf.SetX(colX + colW - w - 4)
	_ = pdf.Cell(nil, text)
}

func writeRow(pdf *gopdf.GoPdf, invoice Invoice, i int, item string, quantity, rate float64, cols colPositions) {
	rowY := pdf.GetY()
	const rowH = 24.0

	// Description
	pdf.SetXY(pageLeft, rowY+8)
	_ = pdf.SetFont("Sans-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	_ = pdf.Cell(nil, truncateToWidth(pdf, item, cols.descWidth-6))

	// Right-aligned numeric / mono cells
	sym := currencySymbol(invoice.Currency)
	amount := strconv.FormatFloat(quantity*rate, 'f', 2, 64)

	if invoice.ShowDateColumn {
		_ = pdf.SetFont("Mono", "", 9.5)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		valueInCol(pdf, getSliceValue(invoice.ItemDates, i), cols.dateX, colWidthDate)
	}
	if invoice.ShowTimeColumn {
		_ = pdf.SetFont("Mono", "", 9.5)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		valueInCol(pdf, getSliceValue(invoice.ItemTimes, i), cols.timeX, colWidthTime)
	}
	if invoice.ShowCategoryColumn {
		_ = pdf.SetFont("Sans", "", 9.5)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		valueInCol(pdf, truncateToWidth(pdf, getSliceValue(invoice.ItemCategories, i), colWidthCategory-6), cols.categoryX, colWidthCategory)
	}
	if invoice.ShowQuantityColumn {
		_ = pdf.SetFont("Mono", "", 10)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		valueInCol(pdf, formatQuantity(quantity), cols.qtyX, colWidthQty)
	}
	if invoice.ShowRateColumn {
		_ = pdf.SetFont("Mono", "", 10)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		valueInCol(pdf, sym+strconv.FormatFloat(rate, 'f', 2, 64), cols.rateX, colWidthRate)
	}
	if invoice.ShowAmountColumn {
		_ = pdf.SetFont("Mono-B", "", 10)
		pdf.SetTextColor(inkR, inkG, inkB)
		valueInCol(pdf, sym+amount, cols.amountX, colWidthAmount)
	}

	// Hairline divider — no row tinting in Swiss
	pdf.SetStrokeColor(hairR, hairG, hairB)
	pdf.SetLineWidth(0.4)
	pdf.Line(pageLeft, rowY+rowH, pageRight, rowY+rowH)

	pdf.SetXY(pageLeft, rowY+rowH)
}

// valueInCol right-aligns a numeric/mono value within its column.
func valueInCol(pdf *gopdf.GoPdf, text string, colX, colW float64) {
	w, _ := pdf.MeasureTextWidth(text)
	pdf.SetY(pdf.GetY()) // y already set by caller
	pdf.SetX(colX + colW - w - 4)
	_ = pdf.Cell(nil, text)
}

// ── Notes section ─────────────────────────────────────────────────────────────

func writeNotes(pdf *gopdf.GoPdf, notes string, startY float64) {
	pdf.SetY(startY)
	pdf.SetX(pageLeft)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "OBSERVAÇÕES")
	pdf.Br(14)
	_ = pdf.SetFont("Sans", "", 9.5)
	pdf.SetTextColor(bdyR, bdyG, bdyB)

	for _, line := range strings.Split(strings.ReplaceAll(notes, `\n`, "\n"), "\n") {
		for _, wrapped := range wrapText(pdf, line, notesMaxWidth) {
			pdf.SetX(pageLeft)
			_ = pdf.Cell(nil, wrapped)
			pdf.Br(14)
		}
	}
}

// ── Exemption reason ──────────────────────────────────────────────────────────

func writeExemptionReason(pdf *gopdf.GoPdf, code, reason, legalReference string) {
	pdf.SetY(695)
	pdf.SetX(pageLeft)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "MOTIVO DE ISENÇÃO DE IVA")
	pdf.Br(14)
	_ = pdf.SetFont("Sans-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	line := code
	if reason != "" {
		if line != "" {
			line += " — " + reason
		} else {
			line = reason
		}
	}
	pdf.SetX(pageLeft)
	_ = pdf.Cell(nil, line)
	pdf.Br(14)
	if legalReference != "" {
		_ = pdf.SetFont("Mono", "", 8)
		pdf.SetTextColor(lblR, lblG, lblB)
		pdf.SetX(pageLeft)
		_ = pdf.Cell(nil, "REF. LEGAL  "+legalReference)
		pdf.Br(13)
	}
}

// ── Totals — flat right-aligned stack, no card ───────────────────────────────

func writeTotals(pdf *gopdf.GoPdf, invoice Invoice, subtotal, tax, discount, taxRate, withholding, withholdingRate, startY float64) {
	pdf.SetY(startY)

	writeTotalLine(pdf, subtotalLabel, subtotal, invoice.Currency, false, false)
	if tax > 0 {
		writeTotalLine(pdf, fmt.Sprintf("%s  %.2f %%", taxLabel, taxRate*100), tax, invoice.Currency, false, false)
	}
	if withholding > 0 {
		writeTotalLine(pdf, fmt.Sprintf("%s  %.2f %%", withholdingLabel, withholdingRate*100), withholding, invoice.Currency, true, false)
	}
	if discount > 0 {
		writeTotalLine(pdf, discountLabel, discount, invoice.Currency, false, false)
	}

	// Ink rule above grand total
	sepY := pdf.GetY() + 4
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(totalsLabelOffset, sepY, totalsRightEdge, sepY)
	pdf.Br(12)

	finalLabel := totalLabel
	if withholding > 0 {
		finalLabel = totalToPayLabel
	}
	writeTotalLine(pdf, finalLabel, subtotal+tax-withholding-discount, invoice.Currency, false, true)

	pdf.Br(6)
}

func writeQuantityTotal(pdf *gopdf.GoPdf, totalQty float64) {
	_ = pdf.SetFont("Mono", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "TOTAL QTD.")

	_ = pdf.SetFont("Mono-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	rightAlignedCell(pdf, formatQuantity(totalQty), totalsRightEdge)
	pdf.Br(22)
}

// writeTotalLine renders one row in the totals stack.
//   - negative: prefix value with "−" and render in red ink
//   - bold:     grand total — ink, Mono-B 26pt for the value
func writeTotalLine(pdf *gopdf.GoPdf, label string, total float64, currency string, negative, bold bool) {
	// Label
	if bold {
		_ = pdf.SetFont("Mono-B", "", 8.5)
	} else {
		_ = pdf.SetFont("Mono", "", 8.5)
	}
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, strings.ToUpper(label))

	// Value — right-aligned to totalsRightEdge
	prefix := ""
	if negative {
		prefix = "−"
	}
	valueStr := prefix + currencySymbol(currency) + strconv.FormatFloat(total, 'f', 2, 64)

	if bold {
		_ = pdf.SetFont("Mono-B", "", 26)
		pdf.SetTextColor(inkR, inkG, inkB)
	} else if negative {
		_ = pdf.SetFont("Mono", "", 11)
		pdf.SetTextColor(negR, negG, negB)
	} else {
		_ = pdf.SetFont("Mono", "", 11)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
	}

	w, _ := pdf.MeasureTextWidth(valueStr)
	if bold {
		// Big number sits slightly above the label baseline
		pdf.SetXY(totalsRightEdge-w, pdf.GetY()-6)
		_ = pdf.Cell(nil, valueStr)
		pdf.Br(34)
	} else {
		pdf.SetX(totalsRightEdge - w)
		_ = pdf.Cell(nil, valueStr)
		pdf.Br(22)
	}
}

func writeDueDate(pdf *gopdf.GoPdf, due string) {
	_ = pdf.SetFont("Mono", "", 8.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "DATA DE VENCIMENTO")

	_ = pdf.SetFont("Mono-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	rightAlignedCell(pdf, due, totalsRightEdge)
	pdf.Br(18)
}

func writePaymentTerms(pdf *gopdf.GoPdf, terms string) {
	_ = pdf.SetFont("Mono", "", 8.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	pdf.SetX(totalsLabelOffset)
	_ = pdf.Cell(nil, "CONDIÇÕES DE PAGAMENTO")

	_ = pdf.SetFont("Mono-B", "", 10)
	pdf.SetTextColor(inkR, inkG, inkB)
	rightAlignedCell(pdf, terms, totalsRightEdge)
	pdf.Br(18)
}

// ── Footer — 3-column mono strip ─────────────────────────────────────────────
// Left:   invoice ID
// Centre: page indicator
// Right:  seller NIF (or empty)

func writeFooter(pdf *gopdf.GoPdf, id string) {
	const y = 812.0

	// Top ink rule
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.5)
	pdf.Line(pageLeft, y-8, pageRight, y-8)

	_ = pdf.SetFont("Mono", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)

	// Left
	pdf.SetXY(pageLeft, y)
	_ = pdf.Cell(nil, id)

	// Centre
	page := "PÁGINA 1 / 1"
	w, _ := pdf.MeasureTextWidth(page)
	pdf.SetXY((pageWidth-w)/2, y)
	_ = pdf.Cell(nil, page)

	// Right — kept generic; main.go can pass seller NIF here in a future iteration.
	right := "CONFORME À AUTORIDADE TRIBUTÁRIA"
	pdf.SetXY(0, y)
	rightAlignedCell(pdf, right, pageRight)
}

// ── Utilities ─────────────────────────────────────────────────────────────────

func getSliceValue(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

func formatQuantity(quantity float64) string {
	return strconv.FormatFloat(quantity, 'f', -1, 64)
}
