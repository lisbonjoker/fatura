package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed "Inter/Inter Variable/Inter.ttf"
var interFont []byte

//go:embed "Inter/Inter Hinted for Windows/Desktop/Inter-Bold.ttf"
var interBoldFont []byte

type Invoice struct {
	Id    string `json:"id" yaml:"id"`
	Title string `json:"title" yaml:"title"`

	Logo string `json:"logo" yaml:"logo"`
	From string `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
	Date string `json:"date" yaml:"date"`
	Due  string `json:"due" yaml:"due"`

	SellerVATID string `json:"seller_vat_id" yaml:"seller_vat_id"`
	BuyerVATID  string `json:"buyer_vat_id" yaml:"buyer_vat_id"`
	CountryCode string `json:"country_code" yaml:"country_code"`

	Items          []string  `json:"items" yaml:"items"`
	ItemDates      []string  `json:"item_dates" yaml:"item_dates"`
	ItemTimes      []string  `json:"item_times" yaml:"item_times"`
	ItemCategories []string  `json:"item_categories" yaml:"item_categories"`
	Quantities     []float64 `json:"quantities" yaml:"quantities"`
	Rates          []float64 `json:"rates" yaml:"rates"`

	ItemColumns        string `json:"item_columns" yaml:"item_columns"`
	ShowDateColumn     bool   `json:"show_date_column" yaml:"show_date_column"`
	ShowTimeColumn     bool   `json:"show_time_column" yaml:"show_time_column"`
	ShowCategoryColumn bool   `json:"show_category_column" yaml:"show_category_column"`
	ShowQuantityColumn bool   `json:"show_quantity_column" yaml:"show_quantity_column"`
	ShowRateColumn     bool   `json:"show_rate_column" yaml:"show_rate_column"`
	ShowAmountColumn   bool   `json:"show_amount_column" yaml:"show_amount_column"`

	Tax      float64 `json:"tax" yaml:"tax"`
	Discount float64 `json:"discount" yaml:"discount"`
	Currency string  `json:"currency" yaml:"currency"`

	ExemptionReason string `json:"exemption_reason" yaml:"exemption_reason"`
	LegalReference  string `json:"legal_reference" yaml:"legal_reference"`
	PaymentTerms    string `json:"payment_terms" yaml:"payment_terms"`
	Note            string `json:"note" yaml:"note"`
}

func DefaultInvoice() Invoice {
	return Invoice{
		Id:          time.Now().Format("20060102"),
		Title:       "INVOICE",
		Rates:       []float64{25},
		Quantities:  []float64{2},
		Items:       []string{"Paper Cranes"},
		From:        "Project Folded, Inc.",
		To:          "Untitled Corporation, Inc.",
		Date:        time.Now().Format("Jan 02, 2006"),
		Due:         time.Now().AddDate(0, 0, 14).Format("Jan 02, 2006"),
		CountryCode: "PT",
		Tax:         0,
		Discount:    0,
		Currency:    "EUR",
		ItemColumns: "date,qty,rate,amount",
	}
}

var (
	importPath     string
	output         string
	fromLines      []string
	toLines        []string
	quantityInput  []string
	file           = Invoice{}
	defaultInvoice = DefaultInvoice()
)

func init() {
	viper.AutomaticEnv()

	generateCmd.Flags().StringVar(&importPath, "import", "", "Imported file (.json/.yaml)")
	generateCmd.Flags().StringVar(&file.Id, "id", time.Now().Format("20060102"), "ID")
	generateCmd.Flags().StringVar(&file.Title, "title", "INVOICE", "Title")

	generateCmd.Flags().Float64SliceVarP(&file.Rates, "rate", "r", defaultInvoice.Rates, "Rates")
	generateCmd.Flags().StringSliceVarP(&quantityInput, "quantity", "q", []string{"2"}, "Quantities (supports decimals, e.g. 0.25)")
	generateCmd.Flags().StringSliceVarP(&file.Items, "item", "i", defaultInvoice.Items, "Items")
	generateCmd.Flags().StringSliceVar(&file.ItemDates, "item-date", nil, "Item dates")
	generateCmd.Flags().StringSliceVar(&file.ItemTimes, "item-time", nil, "Item times (e.g. 08:32-16:43)")
	generateCmd.Flags().StringSliceVar(&file.ItemCategories, "item-category", nil, "Item category/project codes")

	generateCmd.Flags().StringVarP(&file.Logo, "logo", "l", defaultInvoice.Logo, "Company logo")
	generateCmd.Flags().StringVarP(&file.From, "from", "f", defaultInvoice.From, "Issuing company")
	generateCmd.Flags().StringSliceVar(&fromLines, "from-line", nil, "Issuing company line (repeatable)")
	generateCmd.Flags().StringVarP(&file.To, "to", "t", defaultInvoice.To, "Recipient company")
	generateCmd.Flags().StringSliceVar(&toLines, "to-line", nil, "Recipient company line (repeatable)")
	generateCmd.Flags().StringVar(&file.Date, "date", defaultInvoice.Date, "Date")
	generateCmd.Flags().StringVar(&file.Due, "due", defaultInvoice.Due, "Payment due date")
	generateCmd.Flags().StringVar(&file.CountryCode, "country-code", defaultInvoice.CountryCode, "Invoice country code (e.g. PT)")
	generateCmd.Flags().StringVar(&file.SellerVATID, "seller-vat-id", "", "Seller EU VAT ID (e.g. PT123456789)")
	generateCmd.Flags().StringVar(&file.BuyerVATID, "buyer-vat-id", "", "Buyer EU VAT ID")

	generateCmd.Flags().Float64Var(&file.Tax, "tax", defaultInvoice.Tax, "Tax")
	generateCmd.Flags().Float64Var(&file.Tax, "vat", defaultInvoice.Tax, "VAT rate (alias of --tax)")
	generateCmd.Flags().Float64VarP(&file.Discount, "discount", "d", defaultInvoice.Discount, "Discount")
	generateCmd.Flags().StringVarP(&file.Currency, "currency", "c", defaultInvoice.Currency, "Currency")

	generateCmd.Flags().StringVar(&file.ExemptionReason, "exemption-reason", "", "VAT exemption legal reason/code")
	generateCmd.Flags().StringVar(&file.LegalReference, "legal-reference", "", "VAT legal reference (article/code)")
	generateCmd.Flags().StringVar(&ptExemptionPreset, "pt-exemption", "", "PT VAT exemption preset: e_learning|gambling|insurance_financial")
	generateCmd.Flags().StringVar(&file.PaymentTerms, "payment-terms", "", "Payment terms label (e.g. NET 15)")
	generateCmd.Flags().StringVar(&file.ItemColumns, "item-columns", defaultInvoice.ItemColumns, "Comma-separated item columns: date,time,category,qty,rate,amount. Use all or minimal.")
	generateCmd.Flags().StringVarP(&file.Note, "note", "n", "", "Note")
	generateCmd.Flags().StringVarP(&output, "output", "o", "invoice.pdf", "Output file (.pdf)")

	flag.Parse()
}

var rootCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Invoice generates invoices from the command line.",
	Long:  `Invoice generates invoices from the command line.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an invoice",
	Long:  `Generate an invoice`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if importPath != "" {
			err := importData(importPath, &file, cmd.Flags())
			if err != nil {
				return err
			}
		}
		applyPortugueseExemptionPreset(&file)
		applyAddressLines(cmd, &file)
		if err := applyQuantities(cmd, &file); err != nil {
			return err
		}
		applyItemColumnVisibility(&file)
		if err := validateInvoiceCompliance(file); err != nil {
			return err
		}

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{
			PageSize: *gopdf.PageSizeA4,
		})
		pdf.SetMargins(40, 40, 40, 40)
		pdf.AddPage()
		err := pdf.AddTTFFontData("Inter", interFont)
		if err != nil {
			return err
		}

		err = pdf.AddTTFFontData("Inter-Bold", interBoldFont)
		if err != nil {
			return err
		}

		writeLogo(&pdf, file.Logo, file.From)
		writeTitle(&pdf, file.Title, file.Id, file.Date)
		writeBillTo(&pdf, file.To)
		writeRegulatoryDetails(&pdf, file)
		writeHeaderRow(&pdf, file)
		subtotal := 0.0
		totalQty := 0.0
		for i := range file.Items {
			q := 1.0
			if len(file.Quantities) > i {
				q = file.Quantities[i]
			}

			r := 0.0
			if len(file.Rates) > i {
				r = file.Rates[i]
			}

			writeRow(&pdf, file, i, file.Items[i], q, r)
			subtotal += q * r
			totalQty += q
		}
		if file.ExemptionReason != "" {
			writeExemptionReason(&pdf, file.ExemptionReason, file.LegalReference)
		}
		if file.Note != "" {
			writeNotes(&pdf, file.Note)
		}
		writeTotals(&pdf, file, subtotal, subtotal*file.Tax, subtotal*file.Discount, file.Tax, totalQty)
		if file.Due != "" {
			writeDueDate(&pdf, file.Due)
		}
		if file.PaymentTerms != "" {
			writePaymentTerms(&pdf, file.PaymentTerms)
		}
		writeFooter(&pdf, file.Id)
		output = strings.TrimSuffix(output, ".pdf") + ".pdf"
		err = pdf.WritePdf(output)
		if err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", output)

		return nil
	},
}

func applyAddressLines(cmd *cobra.Command, invoice *Invoice) {
	if cmd.Flags().Changed("from-line") && len(fromLines) > 0 {
		invoice.From = strings.Join(fromLines, "\n")
	}
	if cmd.Flags().Changed("to-line") && len(toLines) > 0 {
		invoice.To = strings.Join(toLines, "\n")
	}
}

func applyItemColumnVisibility(invoice *Invoice) {
	columns := strings.ToLower(strings.TrimSpace(invoice.ItemColumns))
	if columns == "" || columns == "all" {
		invoice.ShowDateColumn = true
		invoice.ShowTimeColumn = true
		invoice.ShowCategoryColumn = true
		invoice.ShowQuantityColumn = true
		invoice.ShowRateColumn = true
		invoice.ShowAmountColumn = true
		return
	}
	if columns == "minimal" {
		columns = "qty,amount"
	}
	selected := strings.Split(columns, ",")
	for i := range selected {
		selected[i] = strings.TrimSpace(selected[i])
	}
	has := func(v string) bool {
		return slices.Contains(selected, v)
	}
	invoice.ShowDateColumn = has("date")
	invoice.ShowTimeColumn = has("time")
	invoice.ShowCategoryColumn = has("category")
	invoice.ShowQuantityColumn = has("qty")
	invoice.ShowRateColumn = has("rate")
	invoice.ShowAmountColumn = has("amount")
}

func applyQuantities(cmd *cobra.Command, invoice *Invoice) error {
	if !cmd.Flags().Changed("quantity") {
		return nil
	}

	parsed := make([]float64, 0, len(quantityInput))
	for _, raw := range quantityInput {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			qty, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid quantity %q: expected a number (e.g. 1, 0.5, 0.25)", value)
			}
			parsed = append(parsed, qty)
		}
	}
	if len(parsed) == 0 {
		return fmt.Errorf("at least one non-empty --quantity value is required when --quantity is provided")
	}
	invoice.Quantities = parsed
	return nil
}

func main() {
	rootCmd.AddCommand(generateCmd)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
