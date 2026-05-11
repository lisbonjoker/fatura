package main

import (
	_ "embed"
	"fmt"
	"log"
	"math/rand"
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
	Id    string `json:"id"    yaml:"id"`
	Title string `json:"title" yaml:"title"`

	Logo string `json:"logo" yaml:"logo"`
	From string `json:"from" yaml:"from"`
	To   string `json:"to"   yaml:"to"`
	Date string `json:"date" yaml:"date"`
	Due  string `json:"due"  yaml:"due"`

	SellerVATID string `json:"seller_vat_id" yaml:"seller_vat_id"`
	BuyerVATID  string `json:"buyer_vat_id"  yaml:"buyer_vat_id"`

	Items          []string  `json:"items"            yaml:"items"`
	ItemDates      []string  `json:"item_dates"       yaml:"item_dates"`
	ItemTimes      []string  `json:"item_times"       yaml:"item_times"`
	ItemCategories []string  `json:"item_categories"  yaml:"item_categories"`
	Quantities     []float64 `json:"quantities"       yaml:"quantities"`
	Rates          []float64 `json:"rates"            yaml:"rates"`

	ItemColumns        string `json:"item_columns"         yaml:"item_columns"`
	ShowDateColumn     bool   `json:"show_date_column"     yaml:"show_date_column"`
	ShowTimeColumn     bool   `json:"show_time_column"     yaml:"show_time_column"`
	ShowCategoryColumn bool   `json:"show_category_column" yaml:"show_category_column"`
	ShowQuantityColumn bool   `json:"show_quantity_column" yaml:"show_quantity_column"`
	ShowRateColumn     bool   `json:"show_rate_column"     yaml:"show_rate_column"`
	ShowAmountColumn   bool   `json:"show_amount_column"   yaml:"show_amount_column"`

	Tax      float64 `json:"tax"      yaml:"tax"`
	Discount float64 `json:"discount" yaml:"discount"`
	Currency string  `json:"currency" yaml:"currency"`

	ExemptionCode   string `json:"exemption_code"   yaml:"exemption_code"`
	ExemptionReason string `json:"exemption_reason" yaml:"exemption_reason"`
	LegalReference  string `json:"legal_reference"  yaml:"legal_reference"`
	Reference       string `json:"reference"        yaml:"reference"`
	PaymentTerms    string `json:"payment_terms"    yaml:"payment_terms"`
	Note            string `json:"note"             yaml:"note"`
}

const idChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomSuffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = idChars[rand.Intn(len(idChars))]
	}
	return string(b)
}

func defaultInvoice() Invoice {
	return Invoice{
		Id:          time.Now().Format("20060102") + "-" + randomSuffix(4),
		Title:       "FATURA",
		Rates:       []float64{50},
		Quantities:  []float64{1},
		Items:       []string{"Serviços de Consultoria"},
		From:        "A Minha Empresa, Lda.",
		To:          "Empresa Cliente, S.A.",
		Date:        time.Now().Format("Jan 02, 2006"),
		Due:         time.Now().AddDate(0, 0, 30).Format("Jan 02, 2006"),
		Tax:         0,
		Discount:    0,
		Currency:    "EUR",
		ItemColumns: "date,qty,rate,amount",
	}
}

var (
	importPath    string
	output        string
	fromLines     []string
	toLines       []string
	quantityInput []string
	invoice       = Invoice{}
)

func init() {
	viper.AutomaticEnv()

	d := defaultInvoice()

	generateCmd.Flags().StringVar(&importPath, "import", "", "Ficheiro importado (.json/.yaml)")
	generateCmd.Flags().StringVar(&invoice.Id, "id", d.Id, "Número de fatura")
	generateCmd.Flags().StringVar(&invoice.Title, "title", d.Title, "Título")

	generateCmd.Flags().Float64SliceVarP(&invoice.Rates, "rate", "r", d.Rates, "Preços unitários")
	generateCmd.Flags().StringSliceVarP(&quantityInput, "quantity", "q", []string{"1"}, "Quantidades (suporta decimais, ex: 0.25)")
	generateCmd.Flags().StringSliceVarP(&invoice.Items, "item", "i", d.Items, "Descrição dos artigos/serviços")
	generateCmd.Flags().StringSliceVar(&invoice.ItemDates, "item-date", nil, "Datas dos artigos")
	generateCmd.Flags().StringSliceVar(&invoice.ItemTimes, "item-time", nil, "Horas dos artigos (ex: 09:00-17:00)")
	generateCmd.Flags().StringSliceVar(&invoice.ItemCategories, "item-category", nil, "Categorias/códigos dos artigos")

	generateCmd.Flags().StringVarP(&invoice.Logo, "logo", "l", d.Logo, "Logótipo da empresa")
	generateCmd.Flags().StringVarP(&invoice.From, "from", "f", d.From, "Empresa emissora")
	generateCmd.Flags().StringSliceVar(&fromLines, "from-line", nil, "Linha do emitente (repetível)")
	generateCmd.Flags().StringVarP(&invoice.To, "to", "t", d.To, "Empresa destinatária")
	generateCmd.Flags().StringSliceVar(&toLines, "to-line", nil, "Linha do destinatário (repetível)")
	generateCmd.Flags().StringVar(&invoice.Date, "date", d.Date, "Data de emissão")
	generateCmd.Flags().StringVar(&invoice.Due, "due", d.Due, "Data de vencimento")
	generateCmd.Flags().StringVar(&invoice.SellerVATID, "seller-vat-id", "", "NIF do fornecedor (ex: PT123456789)")
	generateCmd.Flags().StringVar(&invoice.BuyerVATID, "buyer-vat-id", "", "NIF do cliente")

	generateCmd.Flags().Float64Var(&invoice.Tax, "tax", d.Tax, "Taxa de IVA")
	generateCmd.Flags().Float64Var(&invoice.Tax, "iva", d.Tax, "Taxa de IVA (alias de --tax)")
	generateCmd.Flags().Float64VarP(&invoice.Discount, "discount", "d", d.Discount, "Desconto")
	generateCmd.Flags().StringVarP(&invoice.Currency, "currency", "c", d.Currency, "Moeda")

	generateCmd.Flags().StringVar(&exemptionCodeFlag, "exemption", "", "Código de isenção AT (ex: M07, M01, M99). Ver lista completa em --help")
	generateCmd.Flags().StringVar(&invoice.ExemptionReason, "exemption-reason", "", "Motivo de isenção personalizado (substitui o do código AT)")
	generateCmd.Flags().StringVar(&invoice.LegalReference, "legal-reference", "", "Referência legal personalizada (substitui a do código AT)")
	generateCmd.Flags().StringVar(&invoice.Reference, "reference", "", "Referência de encomenda/PO (ex: PO-2026-001)")
	generateCmd.Flags().StringVar(&invoice.PaymentTerms, "payment-terms", "", "Condições de pagamento (ex: 30 dias)")
	generateCmd.Flags().StringVar(&invoice.ItemColumns, "item-columns", d.ItemColumns, "Colunas dos artigos: date,time,category,qty,rate,amount")
	generateCmd.Flags().StringVarP(&invoice.Note, "note", "n", "", "Observações")
	generateCmd.Flags().StringVarP(&output, "output", "o", "fatura.pdf", "Ficheiro de saída (.pdf)")
}

var rootCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Gerador de faturas portuguesas em linha de comandos.",
	Long:  `Gerador de faturas portuguesas em linha de comandos.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Gerar uma fatura",
	Long:  `Gerar uma fatura`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if importPath != "" {
			if err := importData(importPath, &invoice, cmd.Flags()); err != nil {
				return err
			}
		}
		applyExemptionCode(&invoice)
		applyAddressLines(cmd, &invoice)
		if err := applyQuantities(cmd, &invoice); err != nil {
			return err
		}
		applyItemColumnVisibility(&invoice)
		if err := validateInvoiceCompliance(invoice); err != nil {
			return err
		}

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.SetMargins(40, 40, 40, 40)
		pdf.AddPage()

		if err := pdf.AddTTFFontData("Inter", interFont); err != nil {
			return err
		}
		if err := pdf.AddTTFFontData("Inter-Bold", interBoldFont); err != nil {
			return err
		}

		writeLogo(&pdf, invoice.Logo, invoice.From)
		writeTitle(&pdf, invoice.Title, invoice.Id, invoice.Date)
		writeBillTo(&pdf, invoice.To)
		writeRegulatoryDetails(&pdf, invoice)
		writeHeaderRow(&pdf, invoice)

		subtotal := 0.0
		totalQty := 0.0
		for i := range invoice.Items {
			q := 1.0
			if i < len(invoice.Quantities) {
				q = invoice.Quantities[i]
			}
			r := 0.0
			if i < len(invoice.Rates) {
				r = invoice.Rates[i]
			}
			writeRow(&pdf, invoice, i, invoice.Items[i], q, r)
			subtotal += q * r
			totalQty += q
		}

		if invoice.ExemptionCode != "" || invoice.ExemptionReason != "" {
			writeExemptionReason(&pdf, invoice.ExemptionCode, invoice.ExemptionReason, invoice.LegalReference)
		}
		if invoice.Note != "" {
			writeNotes(&pdf, invoice.Note)
		}
		writeTotals(&pdf, invoice, subtotal, subtotal*invoice.Tax, subtotal*invoice.Discount, invoice.Tax, totalQty)
		if invoice.Due != "" {
			writeDueDate(&pdf, invoice.Due)
		}
		if invoice.PaymentTerms != "" {
			writePaymentTerms(&pdf, invoice.PaymentTerms)
		}
		writeFooter(&pdf, invoice.Id)

		output = strings.TrimSuffix(output, ".pdf") + ".pdf"
		if err := pdf.WritePdf(output); err != nil {
			return err
		}
		fmt.Printf("Fatura gerada: %s\n", output)
		return nil
	},
}

func applyAddressLines(cmd *cobra.Command, inv *Invoice) {
	if cmd.Flags().Changed("from-line") && len(fromLines) > 0 {
		inv.From = strings.Join(fromLines, "\n")
	}
	if cmd.Flags().Changed("to-line") && len(toLines) > 0 {
		inv.To = strings.Join(toLines, "\n")
	}
}

func applyItemColumnVisibility(inv *Invoice) {
	columns := strings.ToLower(strings.TrimSpace(inv.ItemColumns))
	if columns == "" || columns == "all" {
		inv.ShowDateColumn = true
		inv.ShowTimeColumn = true
		inv.ShowCategoryColumn = true
		inv.ShowQuantityColumn = true
		inv.ShowRateColumn = true
		inv.ShowAmountColumn = true
		return
	}
	if columns == "minimal" {
		columns = "qty,amount"
	}
	selected := strings.Split(columns, ",")
	for i := range selected {
		selected[i] = strings.TrimSpace(selected[i])
	}
	has := func(v string) bool { return slices.Contains(selected, v) }
	inv.ShowDateColumn = has("date")
	inv.ShowTimeColumn = has("time")
	inv.ShowCategoryColumn = has("category")
	inv.ShowQuantityColumn = has("qty")
	inv.ShowRateColumn = has("rate")
	inv.ShowAmountColumn = has("amount")
}

func applyQuantities(cmd *cobra.Command, inv *Invoice) error {
	if !cmd.Flags().Changed("quantity") {
		return nil
	}
	var parsed []float64
	for _, raw := range quantityInput {
		for _, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			qty, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("quantidade inválida %q: esperado número (ex: 1, 0.5, 0.25)", value)
			}
			parsed = append(parsed, qty)
		}
	}
	if len(parsed) == 0 {
		return fmt.Errorf("é necessário pelo menos uma quantidade válida em --quantity")
	}
	inv.Quantities = parsed
	return nil
}

func main() {
	rootCmd.AddCommand(generateCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
