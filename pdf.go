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
)

const (
	subtotalLabel = "Subtotal"
	discountLabel = "Discount"
	taxLabel      = "Tax"
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

	for i := 0; i < len(fromLines); i++ {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 12)
			_ = pdf.Cell(nil, fromLines[i])
			pdf.Br(18)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
			_ = pdf.Cell(nil, fromLines[i])
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
	_ = pdf.Cell(nil, "Invoice No: ")
	_ = pdf.Cell(nil, id)
	pdf.SetTextColor(150, 150, 150)
	_ = pdf.Cell(nil, "  ·  ")
	pdf.SetTextColor(100, 100, 100)
	_ = pdf.Cell(nil, "Issue Date: ")
	_ = pdf.Cell(nil, date)
	pdf.Br(48)
}

func writeDueDate(pdf *gopdf.GoPdf, due string) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, "Due Date")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(11)
	pdf.SetX(amountColumnOffset - 15)
	_ = pdf.Cell(nil, due)
	pdf.Br(12)
}

func writePaymentTerms(pdf *gopdf.GoPdf, terms string) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, "Payment Terms")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(11)
	pdf.SetX(amountColumnOffset - 15)
	_ = pdf.Cell(nil, terms)
	pdf.Br(12)
}

func writeBillTo(pdf *gopdf.GoPdf, to string) {
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", 9)
	_ = pdf.Cell(nil, "BILL TO")
	pdf.Br(18)
	pdf.SetTextColor(75, 75, 75)

	formattedTo := strings.ReplaceAll(to, `\n`, "\n")
	toLines := strings.Split(formattedTo, "\n")

	for i := 0; i < len(toLines); i++ {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 15)
			_ = pdf.Cell(nil, toLines[i])
			pdf.Br(20)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
			_ = pdf.Cell(nil, toLines[i])
			pdf.Br(15)
		}
	}
	pdf.Br(64)
}

func writeHeaderRow(pdf *gopdf.GoPdf, invoice Invoice) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "ITEM")
	if invoice.ShowDateColumn {
		pdf.SetX(dateColumnOffset)
		_ = pdf.Cell(nil, "DATE")
	}
	if invoice.ShowTimeColumn {
		pdf.SetX(timeColumnOffset)
		_ = pdf.Cell(nil, "TIME")
	}
	if invoice.ShowCategoryColumn {
		pdf.SetX(categoryColumnOffset)
		_ = pdf.Cell(nil, "CATEGORY")
	}
	if invoice.ShowQuantityColumn {
		pdf.SetX(quantityColumnOffset)
		_ = pdf.Cell(nil, "QTY")
	}
	if invoice.ShowRateColumn {
		pdf.SetX(rateColumnOffset)
		_ = pdf.Cell(nil, "RATE")
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(amountColumnOffset)
		_ = pdf.Cell(nil, "AMOUNT")
	}
	pdf.Br(24)
}

func writeRegulatoryDetails(pdf *gopdf.GoPdf, invoice Invoice) {
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", 9)
	_ = pdf.Cell(nil, "REGULATORY DETAILS")
	pdf.Br(16)
	_ = pdf.SetFont("Inter", "", 9.5)
	pdf.SetTextColor(55, 55, 55)

	if invoice.SellerVATID != "" {
		_ = pdf.Cell(nil, "Seller VAT ID: "+invoice.SellerVATID)
		pdf.Br(14)
	}
	if invoice.BuyerVATID != "" {
		_ = pdf.Cell(nil, "Buyer VAT ID: "+invoice.BuyerVATID)
		pdf.Br(14)
	}
	pdf.Br(20)
}

func writeNotes(pdf *gopdf.GoPdf, notes string) {
	pdf.SetY(600)

	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "NOTES")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(0, 0, 0)

	formattedNotes := strings.ReplaceAll(notes, `\n`, "\n")
	notesLines := strings.Split(formattedNotes, "\n")

	for i := 0; i < len(notesLines); i++ {
		_ = pdf.Cell(nil, notesLines[i])
		pdf.Br(15)
	}

	pdf.Br(48)
}

func writeExemptionReason(pdf *gopdf.GoPdf, reason, legalReference string) {
	pdf.SetY(690)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, "VAT EXEMPTION REASON")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, reason)
	pdf.Br(14)
	if legalReference != "" {
		_ = pdf.SetFont("Inter", "", 8.5)
		pdf.SetTextColor(75, 75, 75)
		_ = pdf.Cell(nil, "Legal Ref: "+legalReference)
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

	total := quantity * rate
	amount := strconv.FormatFloat(total, 'f', 2, 64)

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
		_ = pdf.Cell(nil, currencySymbols[file.Currency]+strconv.FormatFloat(rate, 'f', 2, 64))
	}
	if invoice.ShowAmountColumn {
		pdf.SetX(amountColumnOffset)
		_ = pdf.Cell(nil, currencySymbols[file.Currency]+amount)
	}
	pdf.Br(24)
}

func writeTotals(pdf *gopdf.GoPdf, invoice Invoice, subtotal float64, tax float64, discount float64, taxRate float64, totalQty float64) {
	pdf.SetY(600)

	if invoice.ShowQuantityColumn {
		writeQuantityTotal(pdf, totalQty)
	}
	writeTotal(pdf, subtotalLabel, subtotal)
	if tax > 0 {
		writeTotal(pdf, fmt.Sprintf("%s (%.2f%%)", taxLabel, taxRate*100), tax)
	}
	if discount > 0 {
		writeTotal(pdf, discountLabel, discount)
	}
	writeTotal(pdf, totalLabel, subtotal+tax-discount)
}

func writeQuantityTotal(pdf *gopdf.GoPdf, totalQty float64) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, "Total Qty")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(12)
	pdf.SetX(amountColumnOffset - 15)
	_ = pdf.Cell(nil, formatQuantity(totalQty))
	pdf.Br(24)
}

func writeTotal(pdf *gopdf.GoPdf, label string, total float64) {
	_ = pdf.SetFont("Inter", "", 9)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.SetFontSize(12)
	pdf.SetX(amountColumnOffset - 15)
	if label == totalLabel {
		_ = pdf.SetFont("Inter-Bold", "", 11.5)
	}
	_ = pdf.Cell(nil, currencySymbols[file.Currency]+strconv.FormatFloat(total, 'f', 2, 64))
	pdf.Br(24)
}

func getImageDimension(imagePath string) (int, int) {
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	defer file.Close()

	image, _, err := image.DecodeConfig(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", imagePath, err)
	}
	return image.Width, image.Height
}

func getSliceValue(values []string, index int) string {
	if len(values) > index {
		return values[index]
	}
	return ""
}

func formatQuantity(quantity float64) string {
	return strconv.FormatFloat(quantity, 'f', -1, 64)
}
