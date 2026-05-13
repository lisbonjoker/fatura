package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type rptMonth struct {
	subtotal, taxAmount, withholding, total float64
}

type clientSummary struct {
	name     string
	invoices int
	total    float64
	unpaid   float64
}

var ptMonthsShort = [12]string{
	"Jan", "Fev", "Mar", "Abr", "Mai", "Jun",
	"Jul", "Ago", "Set", "Out", "Nov", "Dez",
}

// Column right-edges for the monthly table.
const (
	rptFatR = 220.0 // Faturado
	rptIVAR = 310.0 // IVA
	rptRetR = 425.0 // Retenção IRS
	rptTotR = pageRight
)

// ── Entry point ───────────────────────────────────────────────────────────────

func generateAnnualReportPDF(records []InvoiceRecord, year, compareYear int, currency, outPath string) error {
	months, clients, unpaid, totSub, totTax, totWith, totTotal := aggregateYear(records, year, currency)
	var cmpTotals [12]float64
	cmpYearTotal := 0.0
	if compareYear > 0 {
		for _, r := range records {
			if r.Draft {
				continue
			}
			cur := r.Currency
			if cur == "" {
				cur = "EUR"
			}
			if cur != currency {
				continue
			}
			t, err := time.Parse("Jan 02, 2006", r.Date)
			if err != nil || t.Year() != compareYear {
				continue
			}
			m := int(t.Month()) - 1
			cmpTotals[m] += r.Total
			cmpYearTotal += r.Total
		}
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.SetMargins(40, 40, 40, 40)
	pdf.AddPage()
	for _, reg := range []struct {
		name string
		data []byte
	}{
		{"Sans", plexSansFont},
		{"Sans-B", plexSansBoldFont},
		{"Mono", plexMonoFont},
		{"Mono-B", plexMonoBoldFont},
	} {
		if err := pdf.AddTTFFontData(reg.name, reg.data); err != nil {
			return err
		}
	}

	sym := currencySymbol(currency)
	y := rptHeader(&pdf, year, sym, totTotal)
	y = rptSectionLabel(&pdf, "RESUMO MENSAL", y)
	y = rptMonthlyTable(&pdf, months, sym, totSub, totTax, totWith, totTotal, y)
	y += 12
	y = rptSectionLabel(&pdf, "EVOLUÇÃO MENSAL", y)
	y = rptBarChart(&pdf, months, cmpTotals, compareYear, sym, y)

	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.5)
	pdf.Line(pageLeft, y+6, pageRight, y+6)
	y += 18

	if len(clients) > 0 {
		y = rptSectionLabel(&pdf, "POR CLIENTE", y)
		y = rptClientTable(&pdf, clients, sym, y)
		y += 8
	}

	if len(unpaid) > 0 {
		pdf.SetStrokeColor(divR, divG, divB)
		pdf.SetLineWidth(0.4)
		pdf.Line(pageLeft, y, pageRight, y)
		y += 12
		y = rptSectionLabel(&pdf, "RECEBÍVEIS PENDENTES", y)
		y = rptUnpaidTable(&pdf, unpaid, sym, y)
		y += 8
	}

	if year == time.Now().Year() || (compareYear > 0 && cmpYearTotal > 0) {
		pdf.SetStrokeColor(divR, divG, divB)
		pdf.SetLineWidth(0.4)
		pdf.Line(pageLeft, y, pageRight, y)
		y += 12
		rptStatsRow(&pdf, months, totTotal, year, compareYear, cmpYearTotal, sym, y)
	}

	rptFooter(&pdf, year, currency)
	return pdf.WritePdf(outPath)
}

// aggregateYear groups records for a given year+currency into monthly buckets,
// client summaries, and a list of unpaid invoices.
func aggregateYear(records []InvoiceRecord, year int, currency string) (
	months [12]rptMonth,
	clients []*clientSummary,
	unpaid []InvoiceRecord,
	totSub, totTax, totWith, totTotal float64,
) {
	clientMap := map[string]*clientSummary{}

	for _, r := range records {
		if r.Draft {
			continue
		}
		cur := r.Currency
		if cur == "" {
			cur = "EUR"
		}
		if cur != currency {
			continue
		}
		t, err := time.Parse("Jan 02, 2006", r.Date)
		if err != nil || t.Year() != year {
			continue
		}
		m := int(t.Month()) - 1
		months[m].subtotal += r.Subtotal
		months[m].taxAmount += r.TaxAmount
		months[m].withholding += r.Subtotal * r.Withholding
		months[m].total += r.Total

		name := strings.TrimSpace(strings.Split(strings.ReplaceAll(r.To, `\n`, "\n"), "\n")[0])
		if name == "" {
			name = "—"
		}
		if clientMap[name] == nil {
			clientMap[name] = &clientSummary{name: name}
		}
		clientMap[name].invoices++
		clientMap[name].total += r.Total
		if r.PaidAt == "" {
			clientMap[name].unpaid += r.Total
			unpaid = append(unpaid, r)
		}
	}

	for _, m := range months {
		totSub += m.subtotal
		totTax += m.taxAmount
		totWith += m.withholding
		totTotal += m.total
	}

	clients = make([]*clientSummary, 0, len(clientMap))
	for _, c := range clientMap {
		clients = append(clients, c)
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].total > clients[j].total
	})
	return
}

// ── Section helpers ───────────────────────────────────────────────────────────

func rptHeader(pdf *gopdf.GoPdf, year int, sym string, totTotal float64) float64 {
	const stripY = 36.0

	pdf.SetFillColor(accR, accG, accB)
	pdf.RectFromUpperLeftWithStyle(pageLeft, stripY+3, 8, 8, "F")

	pdf.SetXY(pageLeft+14, stripY)
	_ = pdf.SetFont("Sans-B", "", 11)
	pdf.SetTextColor(inkR, inkG, inkB)
	_ = pdf.Cell(nil, expandLetterSpacing(fmt.Sprintf("RELATÓRIO ANUAL %d", year)))

	pdf.SetXY(0, stripY)
	_ = pdf.SetFont("Mono", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)
	rightAlignedCell(pdf, "TOTAL ANUAL", pageRight)

	pdf.SetXY(0, stripY+12)
	_ = pdf.SetFont("Mono-B", "", 18)
	pdf.SetTextColor(inkR, inkG, inkB)
	rightAlignedCell(pdf, sym+strconv.FormatFloat(totTotal, 'f', 2, 64), pageRight)

	ruleY := stripY + 34
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(1)
	pdf.Line(pageLeft, ruleY, pageRight, ruleY)

	return ruleY + 16
}

func rptSectionLabel(pdf *gopdf.GoPdf, label string, y float64) float64 {
	pdf.SetXY(pageLeft, y)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, label)
	return y + 14
}

func rptMonthlyTable(pdf *gopdf.GoPdf, months [12]rptMonth, sym string, totSub, totTax, totWith, totTotal, y float64) float64 {
	const rowH = 15.0

	// Header
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(pageLeft, y, pageRight, y)

	pdf.SetXY(pageLeft, y+5)
	_ = pdf.SetFont("Mono", "", 7.5)
	pdf.SetTextColor(lblR, lblG, lblB)
	_ = pdf.Cell(nil, "MÊS")
	for _, col := range []struct {
		label string
		right float64
	}{
		{"FATURADO", rptFatR},
		{"IVA", rptIVAR},
		{"RETENÇÃO IRS", rptRetR},
		{"TOTAL", rptTotR},
	} {
		pdf.SetXY(0, y+5)
		rightAlignedCell(pdf, col.label, col.right)
	}
	y += 18

	// Month rows
	for i, name := range ptMonths {
		pdf.SetStrokeColor(hairR, hairG, hairB)
		pdf.SetLineWidth(0.3)
		pdf.Line(pageLeft, y, pageRight, y)

		active := months[i].total > 0
		pdf.SetXY(pageLeft, y+4)
		if active {
			_ = pdf.SetFont("Sans-B", "", 9)
			pdf.SetTextColor(inkR, inkG, inkB)
		} else {
			_ = pdf.SetFont("Sans", "", 9)
			pdf.SetTextColor(lblR, lblG, lblB)
		}
		_ = pdf.Cell(nil, name)

		_ = pdf.SetFont("Mono", "", 9)
		if active {
			pdf.SetTextColor(bdyR, bdyG, bdyB)
		} else {
			pdf.SetTextColor(lblR, lblG, lblB)
		}
		for _, col := range []struct {
			val   float64
			right float64
			bold  bool
		}{
			{months[i].subtotal, rptFatR, false},
			{months[i].taxAmount, rptIVAR, false},
			{months[i].withholding, rptRetR, false},
			{months[i].total, rptTotR, true},
		} {
			pdf.SetXY(0, y+4)
			if col.bold && active {
				_ = pdf.SetFont("Mono-B", "", 9)
				pdf.SetTextColor(inkR, inkG, inkB)
			} else {
				_ = pdf.SetFont("Mono", "", 9)
			}
			rightAlignedCell(pdf, sym+strconv.FormatFloat(col.val, 'f', 2, 64), col.right)
		}
		y += rowH
	}

	// Total row
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.8)
	pdf.Line(pageLeft, y, pageRight, y)
	y += 4

	pdf.SetXY(pageLeft, y+4)
	_ = pdf.SetFont("Mono-B", "", 9)
	pdf.SetTextColor(inkR, inkG, inkB)
	_ = pdf.Cell(nil, "TOTAL")

	for _, col := range []struct {
		val   float64
		right float64
	}{
		{totSub, rptFatR},
		{totTax, rptIVAR},
		{totWith, rptRetR},
		{totTotal, rptTotR},
	} {
		pdf.SetXY(0, y+4)
		rightAlignedCell(pdf, sym+strconv.FormatFloat(col.val, 'f', 2, 64), col.right)
	}
	return y + 18
}

func rptBarChart(pdf *gopdf.GoPdf, months [12]rptMonth, cmpTotals [12]float64, compareYear int, sym string, y float64) float64 {
	const maxBarH = 60.0
	const chartH = maxBarH + 16.0 // + x-axis labels

	slotW := (pageRight - pageLeft) / 12.0
	hasCompare := compareYear > 0

	// Find max for scaling
	maxVal := 0.0
	for i, m := range months {
		if m.total > maxVal {
			maxVal = m.total
		}
		if hasCompare && cmpTotals[i] > maxVal {
			maxVal = cmpTotals[i]
		}
	}

	baseY := y + maxBarH

	for i, m := range months {
		slotX := pageLeft + float64(i)*slotW

		if hasCompare {
			// Side-by-side: grey (compare) left, orange (current) right
			barW := slotW * 0.38
			leftX := slotX + slotW*0.07
			rightX := slotX + slotW*0.07 + barW + slotW*0.08

			// Compare bar (grey)
			if maxVal > 0 && cmpTotals[i] > 0 {
				h := (cmpTotals[i] / maxVal) * maxBarH
				pdf.SetFillColor(200, 200, 200)
				pdf.RectFromUpperLeftWithStyle(leftX, baseY-h, barW, h, "F")
			}
			// Current bar (orange)
			if maxVal > 0 && m.total > 0 {
				h := (m.total / maxVal) * maxBarH
				pdf.SetFillColor(accR, accG, accB)
				pdf.RectFromUpperLeftWithStyle(rightX, baseY-h, barW, h, "F")
			}
		} else {
			barW := slotW * 0.55
			barX := slotX + (slotW-barW)/2
			if maxVal > 0 && m.total > 0 {
				h := (m.total / maxVal) * maxBarH
				pdf.SetFillColor(accR, accG, accB)
				pdf.RectFromUpperLeftWithStyle(barX, baseY-h, barW, h, "F")
				// Value above bar
				if h > 10 {
					pdf.SetXY(0, baseY-h-10)
					_ = pdf.SetFont("Mono", "", 6)
					pdf.SetTextColor(inkR, inkG, inkB)
					label := sym + strconv.FormatFloat(m.total, 'f', 0, 64)
					lw, _ := pdf.MeasureTextWidth(label)
					pdf.SetX(barX + (barW-lw)/2)
					_ = pdf.Cell(nil, label)
				}
			}
		}

		// Month label below baseline
		pdf.SetXY(0, baseY+4)
		_ = pdf.SetFont("Mono", "", 6.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		label := ptMonthsShort[i]
		lw, _ := pdf.MeasureTextWidth(label)
		pdf.SetX(slotX + (slotW-lw)/2)
		_ = pdf.Cell(nil, label)
	}

	// Baseline rule
	pdf.SetStrokeColor(divR, divG, divB)
	pdf.SetLineWidth(0.4)
	pdf.Line(pageLeft, baseY, pageRight, baseY)

	// Legend when comparing
	if hasCompare {
		legendY := baseY + 14
		pdf.SetFillColor(200, 200, 200)
		pdf.RectFromUpperLeftWithStyle(pageLeft, legendY, 8, 8, "F")
		pdf.SetXY(pageLeft+11, legendY)
		_ = pdf.SetFont("Mono", "", 7)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, strconv.Itoa(compareYear))

		pdf.SetFillColor(accR, accG, accB)
		pdf.RectFromUpperLeftWithStyle(pageLeft+50, legendY, 8, 8, "F")
		pdf.SetXY(pageLeft+63, legendY)
		_ = pdf.Cell(nil, strconv.Itoa(compareYear+1))
		return y + chartH + 24
	}

	return y + chartH + 8
}

func rptClientTable(pdf *gopdf.GoPdf, clients []*clientSummary, sym string, y float64) float64 {
	const rowH = 16.0
	const maxClients = 8

	for i, c := range clients {
		if i >= maxClients {
			pdf.SetXY(pageLeft, y+3)
			_ = pdf.SetFont("Mono", "", 7.5)
			pdf.SetTextColor(lblR, lblG, lblB)
			_ = pdf.Cell(nil, fmt.Sprintf("… e mais %d cliente(s)", len(clients)-maxClients))
			y += 14
			break
		}

		pdf.SetXY(pageLeft, y+4)
		_ = pdf.SetFont("Sans-B", "", 9.5)
		pdf.SetTextColor(inkR, inkG, inkB)
		_ = pdf.Cell(nil, truncateToWidth(pdf, c.name, 240))

		// Invoice count
		inv := "fatura"
		if c.invoices != 1 {
			inv = "faturas"
		}
		pdf.SetXY(260, y+4)
		_ = pdf.SetFont("Mono", "", 8.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, fmt.Sprintf("%d %s", c.invoices, inv))

		// Total
		pdf.SetXY(0, y+4)
		_ = pdf.SetFont("Mono-B", "", 9.5)
		pdf.SetTextColor(inkR, inkG, inkB)
		rightAlignedCell(pdf, sym+strconv.FormatFloat(c.total, 'f', 2, 64), pageRight)

		// Unpaid sub-label
		if c.unpaid > 0 {
			pdf.SetXY(0, y+4)
			_ = pdf.SetFont("Mono", "", 7.5)
			pdf.SetTextColor(negR, negG, negB)
			rightAlignedCell(pdf, fmt.Sprintf("pendente %s%.2f", sym, c.unpaid), pageRight-100)
		}

		pdf.SetStrokeColor(hairR, hairG, hairB)
		pdf.SetLineWidth(0.3)
		pdf.Line(pageLeft, y+rowH, pageRight, y+rowH)
		y += rowH
	}
	return y
}

func rptUnpaidTable(pdf *gopdf.GoPdf, unpaid []InvoiceRecord, sym string, y float64) float64 {
	const rowH = 15.0
	const maxUnpaid = 6

	for i, r := range unpaid {
		if i >= maxUnpaid {
			pdf.SetXY(pageLeft, y+3)
			_ = pdf.SetFont("Mono", "", 7.5)
			pdf.SetTextColor(lblR, lblG, lblB)
			_ = pdf.Cell(nil, fmt.Sprintf("… e mais %d fatura(s) pendente(s)", len(unpaid)-maxUnpaid))
			y += 13
			break
		}

		pdf.SetXY(pageLeft, y+4)
		_ = pdf.SetFont("Mono-B", "", 8.5)
		pdf.SetTextColor(inkR, inkG, inkB)
		_ = pdf.Cell(nil, r.Id)

		pdf.SetXY(120, y+4)
		_ = pdf.SetFont("Sans", "", 8.5)
		pdf.SetTextColor(bdyR, bdyG, bdyB)
		client := strings.TrimSpace(strings.Split(strings.ReplaceAll(r.To, `\n`, "\n"), "\n")[0])
		_ = pdf.Cell(nil, truncateToWidth(pdf, client, 200))

		pdf.SetXY(330, y+4)
		_ = pdf.SetFont("Mono", "", 8.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, r.Date)

		pdf.SetXY(0, y+4)
		_ = pdf.SetFont("Mono-B", "", 8.5)
		pdf.SetTextColor(negR, negG, negB)
		rightAlignedCell(pdf, sym+strconv.FormatFloat(r.Total, 'f', 2, 64), pageRight)

		pdf.SetStrokeColor(hairR, hairG, hairB)
		pdf.SetLineWidth(0.3)
		pdf.Line(pageLeft, y+rowH, pageRight, y+rowH)
		y += rowH
	}
	return y
}

func rptStatsRow(pdf *gopdf.GoPdf, months [12]rptMonth, totTotal float64, year, compareYear int, cmpYearTotal float64, sym string, y float64) {
	col := pageLeft
	colW := (pageRight - pageLeft) / 2.0

	// Projection (left box) — only for current year
	if year == time.Now().Year() {
		monthsElapsed := int(time.Now().Month())
		if totTotal > 0 && monthsElapsed > 0 {
			projected := totTotal / float64(monthsElapsed) * 12
			pdf.SetXY(col, y)
			_ = pdf.SetFont("Mono", "", 7.5)
			pdf.SetTextColor(lblR, lblG, lblB)
			_ = pdf.Cell(nil, fmt.Sprintf("PROJEÇÃO ANUAL (%d MESES)", monthsElapsed))
			pdf.SetXY(col, y+13)
			_ = pdf.SetFont("Mono-B", "", 13)
			pdf.SetTextColor(inkR, inkG, inkB)
			_ = pdf.Cell(nil, sym+strconv.FormatFloat(projected, 'f', 2, 64))
		}
	}

	// YoY comparison (right box)
	if compareYear > 0 && cmpYearTotal > 0 {
		delta := (totTotal - cmpYearTotal) / cmpYearTotal * 100
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		col2 := col + colW + 10
		pdf.SetXY(col2, y)
		_ = pdf.SetFont("Mono", "", 7.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, fmt.Sprintf("VS %d", compareYear))
		pdf.SetXY(col2, y+13)
		_ = pdf.SetFont("Mono-B", "", 13)
		if delta >= 0 {
			pdf.SetTextColor(inkR, inkG, inkB)
		} else {
			pdf.SetTextColor(negR, negG, negB)
		}
		_ = pdf.Cell(nil, fmt.Sprintf("%s%.1f%%", sign, delta))
		pdf.SetXY(col2, y+28)
		_ = pdf.SetFont("Mono", "", 7.5)
		pdf.SetTextColor(lblR, lblG, lblB)
		_ = pdf.Cell(nil, fmt.Sprintf("%d %s%.2f  →  %d %s%.2f",
			compareYear, sym, cmpYearTotal, year, sym, totTotal))
	}
}

func rptFooter(pdf *gopdf.GoPdf, year int, currency string) {
	const y = 812.0
	pdf.SetStrokeColor(inkR, inkG, inkB)
	pdf.SetLineWidth(0.5)
	pdf.Line(pageLeft, y-8, pageRight, y-8)

	_ = pdf.SetFont("Mono", "", 8)
	pdf.SetTextColor(lblR, lblG, lblB)

	pdf.SetXY(pageLeft, y)
	_ = pdf.Cell(nil, fmt.Sprintf("RELATÓRIO ANUAL %d", year))

	centre := fmt.Sprintf("%s · GERADO EM %s", currency, time.Now().Format("02/01/2006"))
	cw, _ := pdf.MeasureTextWidth(centre)
	pdf.SetXY((pageWidth-cw)/2, y)
	_ = pdf.Cell(nil, centre)

	right := "FATURA CLI"
	pdf.SetXY(0, y)
	rightAlignedCell(pdf, right, pageRight)
}
