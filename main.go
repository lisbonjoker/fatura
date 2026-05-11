package main

import (
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildVersion is overridden at build time via:
//
//	go build -ldflags "-X main.buildVersion=v1.2.3"
var buildVersion = ""

func getVersion() string {
	if buildVersion != "" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

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

	Items          []string  `json:"items"           yaml:"items"`
	ItemDates      []string  `json:"item_dates"      yaml:"item_dates"`
	ItemTimes      []string  `json:"item_times"      yaml:"item_times"`
	ItemCategories []string  `json:"item_categories" yaml:"item_categories"`
	Quantities     []float64 `json:"quantities"      yaml:"quantities"`
	Rates          []float64 `json:"rates"           yaml:"rates"`

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
	ATCUDCode       string `json:"atcud_code"       yaml:"atcud_code"`
	PaymentTerms    string `json:"payment_terms"    yaml:"payment_terms"`
	Withholding     float64 `json:"withholding"    yaml:"withholding"`
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
	clientName    string
	recurName     string
	draft         bool
	invoice       = Invoice{}
)

func init() {
	viper.AutomaticEnv()

	d := defaultInvoice()

	// generate
	generateCmd.Flags().StringVar(&importPath, "import", "", "Ficheiro importado (.json/.yaml)")
	generateCmd.Flags().StringVar(&invoice.Id, "id", "", "Número de fatura (gerado automaticamente se omitido)")
	generateCmd.Flags().StringVar(&invoice.Title, "title", d.Title, "Título")
	generateCmd.Flags().StringVar(&clientName, "client", "", "Nome do cliente em ~/.invoice/clients/<nome>.yaml")
	generateCmd.Flags().StringVar(&recurName, "recur", "", "Guardar como modelo recorrente com este nome")
	generateCmd.Flags().BoolVar(&draft, "draft", false, "Gerar rascunho com marca de água RASCUNHO (não incrementa contador)")

	generateCmd.Flags().Float64SliceVarP(&invoice.Rates, "rate", "r", d.Rates, "Preços unitários")
	generateCmd.Flags().StringSliceVarP(&quantityInput, "quantity", "q", []string{"1"}, "Quantidades (suporta decimais, ex: 0.25)")
	generateCmd.Flags().StringSliceVarP(&invoice.Items, "item", "i", d.Items, "Descrição dos artigos/serviços")
	generateCmd.Flags().StringArrayVar(&invoice.ItemDates, "item-date", nil, "Datas dos artigos")
	generateCmd.Flags().StringArrayVar(&invoice.ItemTimes, "item-time", nil, "Horas dos artigos (ex: 09:00-17:00)")
	generateCmd.Flags().StringArrayVar(&invoice.ItemCategories, "item-category", nil, "Categorias/códigos dos artigos")

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

	generateCmd.Flags().StringVar(&exemptionCodeFlag, "exemption", "", "Código de isenção AT (ex: M07, M01, M99)")
	generateCmd.Flags().StringVar(&invoice.ExemptionReason, "exemption-reason", "", "Motivo de isenção personalizado")
	generateCmd.Flags().StringVar(&invoice.LegalReference, "legal-reference", "", "Referência legal personalizada")
	generateCmd.Flags().StringVar(&invoice.Reference, "reference", "", "Referência de encomenda/PO (ex: PO-2026-001)")
	generateCmd.Flags().StringVar(&invoice.ATCUDCode, "atcud-code", "", "Código de validação ATCUD (obtido no portal AT)")
	generateCmd.Flags().StringVar(&invoice.PaymentTerms, "payment-terms", "", "Condições de pagamento (ex: 30 dias)")
	generateCmd.Flags().Float64Var(&invoice.Withholding, "withholding", 0, "Taxa de retenção na fonte IRS (ex: 0.25 para 25%)")
	generateCmd.Flags().StringVar(&invoice.ItemColumns, "item-columns", d.ItemColumns, "Colunas dos artigos: date,time,category,qty,rate,amount")
	generateCmd.Flags().StringVarP(&invoice.Note, "note", "n", "", "Observações")
	generateCmd.Flags().StringVarP(&output, "output", "o", "fatura.pdf", "Ficheiro de saída (.pdf)")

	// send
	sendCmd.Flags().StringP("to", "t", "", "Endereço de email do destinatário")
	sendCmd.Flags().StringP("pdf", "p", "", "Caminho para o ficheiro PDF")
	sendCmd.Flags().StringP("subject", "s", "", "Assunto do email")
	_ = sendCmd.MarkFlagRequired("to")
	_ = sendCmd.MarkFlagRequired("pdf")

	rootCmd.AddCommand(generateCmd, listCmd, showCmd, sendCmd, versionCmd)

	// Remove the shell completion command — not needed for end users.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Replace the built-in help subcommand with a Portuguese version.
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:                   "help [comando]",
		Short:                 "Ajuda sobre qualquer comando",
		Long:                  `Apresenta informação de ajuda sobre qualquer comando.`,
		DisableFlagsInUseLine: true,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, err := c.Root().Find(args)
			if cmd == nil || err != nil {
				c.Printf("Comando desconhecido: %q\n", args)
			} else {
				_ = cmd.Help()
			}
		},
	})

	// Portuguese usage template — translates all Cobra section headers.
	rootCmd.SetUsageTemplate(`Utilização:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Exemplos:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Comandos disponíveis:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Opções:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasAvailableInheritedFlags}}

Opções globais:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Tópicos de ajuda adicionais:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Utilize "{{.CommandPath}} [comando] --help" para mais informação sobre um comando.{{end}}
`)

	// Translate the --help flag description on every command.
	for _, cmd := range []*cobra.Command{rootCmd, generateCmd, listCmd, showCmd, sendCmd, versionCmd} {
		cmd.InitDefaultHelpFlag()
		if f := cmd.Flags().Lookup("help"); f != nil {
			f.Usage = "Mostrar esta ajuda"
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "fatura",
	Short: "Gerador de faturas portuguesas em linha de comandos.",
	Long: `fatura — Gerador de faturas portuguesas em linha de comandos.

Gera PDFs de faturas conformes com os requisitos da Autoridade Tributária (AT),
incluindo numeração sequencial automática, isenções de IVA (M01–M99), ATCUD,
NIF de emitente/cliente e envio por email.

Comandos disponíveis:
  generate   Gerar um PDF de fatura
  list       Listar faturas emitidas
  show       Mostrar detalhes de uma fatura
  send       Enviar fatura por email
  version    Mostrar a versão instalada

Exemplos:
  fatura generate --from "Empresa, Lda." --to "Cliente, S.A." \
    --item "Consultoria" --rate 500 --iva 0.23 \
    --seller-vat-id PT501234567 --buyer-vat-id PT509876543

  fatura generate --client empresa-xpto --item "Manutenção" --rate 200 --iva 0.23

  fatura list
  fatura show INV-2026-001
  fatura send --to cliente@exemplo.pt --pdf fatura.pdf

Configuração guardada em ~/.invoice/ (histórico, clientes, modelos, SMTP).`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Gerar uma fatura em PDF",
	Long: `Gera um PDF de fatura e guarda-o automaticamente em:
  ~/.invoice/history/<cliente>/<ano>/<mês>/<id>-<cliente>.pdf

O número de fatura é atribuído automaticamente no formato INV-YYYY-NNN
(ex: INV-2026-001). O contador anual é guardado em ~/.invoice/counter.json.

Com --draft, é gerado um rascunho com marca de água "RASCUNHO" sem consumir
um número do contador.

Configuração de cliente:
  Guarde dados recorrentes em ~/.invoice/clients/<nome>.yaml e carregue-os
  com --client <nome>. Os flags da linha de comandos têm sempre precedência.

Isenção de IVA:
  Use --exemption <código> com um código AT (ex: M07, M09). O motivo e a
  referência legal correspondentes são preenchidos automaticamente.

ATCUD:
  Obtenha o código de validação no portal AT e indique-o com --atcud-code.
  O ATCUD aparece na fatura como <código>-<número sequencial>.

Exemplos:
  fatura generate --from "Empresa, Lda." --to "Cliente, S.A." \
    --item "Serviços de consultoria" --quantity 8 --rate 75 \
    --iva 0.23 --seller-vat-id PT501234567 --buyer-vat-id PT509876543

  fatura generate --client empresa-xpto \
    --item "Desenvolvimento" --quantity 20 --rate 90 --iva 0.23

  fatura generate --import fatura.yaml --note "Ref. maio 2026."

  fatura generate --item "Formação" --rate 1200 --exemption M09 \
    --seller-vat-id PT501234567

  fatura generate --item "Desenvolvimento" --rate 100 --quantity 10 \
    --iva 0.23 --withholding 0.25

  fatura generate --draft --from "Empresa" --to "Cliente" \
    --item "Teste" --rate 100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load client config first (lowest priority).
		if clientName != "" {
			loaded, err := loadClientConfig(clientName)
			if err != nil {
				return err
			}
			mergeInvoice(&invoice, loaded)
		}

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

		// Assign invoice number: auto-increment for real invoices, DRAFT prefix for drafts.
		if invoice.Id == "" {
			if draft {
				invoice.Id = "RASCUNHO-" + time.Now().Format("20060102") + "-" + randomSuffix(4)
			} else {
				id, err := nextInvoiceNumber()
				if err != nil {
					return err
				}
				invoice.Id = id
			}
		}

		// Build ATCUD string: VALIDATION_CODE-SEQUENCE (if validation code supplied).
		atcud := ""
		if invoice.ATCUDCode != "" {
			seq := strings.TrimPrefix(invoice.Id, fmt.Sprintf("INV-%d-", time.Now().Year()))
			atcud = invoice.ATCUDCode + "-" + seq
		}

		if !draft {
			if err := validateInvoiceCompliance(invoice); err != nil {
				return err
			}
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

		if draft {
			writeDraftWatermark(&pdf)
		}

		writeHeader(&pdf, invoice, atcud)
		writeInfoStrip(&pdf, invoice)
		cols := computeColPositions(invoice)
		writeHeaderRow(&pdf, invoice, cols)

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
			writeRow(&pdf, invoice, i, invoice.Items[i], q, r, cols)
			subtotal += q * r
			totalQty += q
		}

		// Dynamic section Y: follow items closely, but never overlap the info strip.
		sectionY := pdf.GetY() + 20
		if sectionY < 490 {
			sectionY = 490
		}

		if invoice.ExemptionCode != "" || invoice.ExemptionReason != "" {
			writeExemptionReason(&pdf, invoice.ExemptionCode, invoice.ExemptionReason, invoice.LegalReference)
		}
		if invoice.Note != "" {
			writeNotes(&pdf, invoice.Note, sectionY)
		}
		writeTotals(&pdf, invoice, subtotal, subtotal*invoice.Tax, subtotal*invoice.Discount, invoice.Tax, subtotal*invoice.Withholding, invoice.Withholding, sectionY)
		if invoice.Due != "" {
			writeDueDate(&pdf, invoice.Due)
		}
		if invoice.PaymentTerms != "" {
			writePaymentTerms(&pdf, invoice.PaymentTerms)
		}
		writeFooter(&pdf, invoice.Id)

		// Use auto path unless --output was explicitly provided.
		if !cmd.Flags().Changed("output") {
			autoPath, err := invoicePDFPath(invoice)
			if err != nil {
				return err
			}
			output = autoPath
		} else {
			output = strings.TrimSuffix(output, ".pdf") + ".pdf"
		}

		if err := pdf.WritePdf(output); err != nil {
			return err
		}

		total := subtotal + subtotal*invoice.Tax - subtotal*invoice.Withholding - subtotal*invoice.Discount
		if !draft {
			_ = saveToHistory(invoice, output, total, false)
		}

		if recurName != "" {
			if err := saveRecurringTemplate(recurName, invoice); err != nil {
				fmt.Fprintf(os.Stderr, "aviso: não foi possível guardar modelo recorrente: %v\n", err)
			} else {
				fmt.Printf("Modelo recorrente guardado: ~/.invoice/recurring/%s.yaml\n", recurName)
			}
		}

		label := "Fatura gerada"
		if draft {
			label = "Rascunho gerado"
		}
		fmt.Printf("%s: %s\n", label, output)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Listar faturas emitidas",
	Long: `Lista todas as faturas guardadas no histórico (~/.invoice/history.json).

Mostra ID, data, cliente, total e caminho do ficheiro PDF para cada fatura.
Os rascunhos são assinalados com [rascunho].

Exemplo:
  fatura list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := loadHistory()
		if err != nil {
			return err
		}
		if len(records) == 0 {
			fmt.Println("Nenhuma fatura emitida.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tDATA\tCLIENTE\tTOTAL\tFICHEIRO")
		for _, r := range records {
			label := r.Id
			if r.Draft {
				label += " [rascunho]"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s%.2f\t%s\n",
				label, r.Date, truncate(r.To, 30),
				currencySymbol(r.Currency), r.Total, r.PDF)
		}
		return w.Flush()
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Mostrar detalhes de uma fatura",
	Long: `Mostra os detalhes completos de uma fatura a partir do seu ID.

Exemplo:
  fatura show INV-2026-001`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		records, err := loadHistory()
		if err != nil {
			return err
		}
		for _, r := range records {
			if r.Id == id {
				fmt.Printf("ID:         %s\n", r.Id)
				fmt.Printf("Cliente:    %s\n", r.To)
				fmt.Printf("Data:       %s\n", r.Date)
				fmt.Printf("Total:      %s%.2f\n", currencySymbol(r.Currency), r.Total)
				fmt.Printf("Ficheiro:   %s\n", r.PDF)
				fmt.Printf("Emitida em: %s\n", r.IssuedAt)
				if r.Draft {
					fmt.Println("Tipo:       Rascunho")
				}
				return nil
			}
		}
		return fmt.Errorf("fatura %q não encontrada", id)
	},
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Enviar fatura por email",
	Long: `Envia um PDF de fatura por email utilizando as credenciais SMTP em
~/.invoice/config.yaml.

Configuração necessária (~/.invoice/config.yaml):
  smtp:
    host: smtp.exemplo.pt
    port: 587
    user: utilizador@exemplo.pt
    password: palavra-passe
    from: Empresa <faturacao@exemplo.pt>

Suporta STARTTLS (porta 587) e TLS implícito (porta 465).

Exemplos:
  fatura send --to cliente@exemplo.pt --pdf fatura.pdf
  fatura send --to cliente@exemplo.pt --pdf fatura.pdf --subject "Fatura INV-2026-001"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		pdfPath, _ := cmd.Flags().GetString("pdf")
		subject, _ := cmd.Flags().GetString("subject")
		if subject == "" {
			subject = "Fatura"
		}
		cfg, err := loadGlobalConfig()
		if err != nil {
			return err
		}
		if err := sendInvoiceEmail(to, subject, pdfPath, cfg.SMTP); err != nil {
			return err
		}
		fmt.Printf("Fatura enviada para %s\n", to)
		return nil
	},
}

// mergeInvoice copies non-zero fields from src into dst (dst takes precedence).
func mergeInvoice(dst *Invoice, src Invoice) {
	if dst.To == "" || dst.To == defaultInvoice().To {
		dst.To = src.To
	}
	if dst.BuyerVATID == "" {
		dst.BuyerVATID = src.BuyerVATID
	}
	if dst.From == "" || dst.From == defaultInvoice().From {
		dst.From = src.From
	}
	if dst.SellerVATID == "" {
		dst.SellerVATID = src.SellerVATID
	}
	if dst.Currency == "" {
		dst.Currency = src.Currency
	}
	if dst.PaymentTerms == "" {
		dst.PaymentTerms = src.PaymentTerms
	}
	if dst.Note == "" {
		dst.Note = src.Note
	}
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostrar a versão instalada",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("invoice " + getVersion())
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
