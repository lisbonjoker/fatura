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

//go:embed "IBMPlex/IBMPlexSans-Regular.ttf"
var plexSansFont []byte

//go:embed "IBMPlex/IBMPlexSans-SemiBold.ttf"
var plexSansBoldFont []byte

//go:embed "IBMPlex/IBMPlexMono-Regular.ttf"
var plexMonoFont []byte

//go:embed "IBMPlex/IBMPlexMono-Medium.ttf"
var plexMonoBoldFont []byte

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

	ExemptionCode   string  `json:"exemption_code"   yaml:"exemption_code"`
	ExemptionReason string  `json:"exemption_reason" yaml:"exemption_reason"`
	LegalReference  string  `json:"legal_reference"  yaml:"legal_reference"`
	Reference       string  `json:"reference"        yaml:"reference"`
	ATCUDCode       string  `json:"atcud_code"       yaml:"atcud_code"`
	PaymentTerms    string  `json:"payment_terms"    yaml:"payment_terms"`
	Withholding     float64 `json:"withholding"    yaml:"withholding"`
	Note            string  `json:"note"             yaml:"note"`
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

	// summary
	summaryCmd.Flags().String("pdf", "", "Gerar relatório anual em PDF para o ficheiro indicado")
	summaryCmd.Flags().Int("compare", 0, "Comparar com o ano indicado (ex: --compare 2025)")

	// send
	sendCmd.Flags().StringP("to", "t", "", "Endereço de email do destinatário")
	sendCmd.Flags().StringP("pdf", "p", "", "Caminho para o ficheiro PDF")
	sendCmd.Flags().StringP("subject", "s", "", "Assunto do email")
	_ = sendCmd.MarkFlagRequired("to")
	_ = sendCmd.MarkFlagRequired("pdf")

	rootCmd.AddCommand(generateCmd, listCmd, showCmd, sendCmd, summaryCmd, payCmd, versionCmd)

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
	for _, cmd := range []*cobra.Command{rootCmd, generateCmd, listCmd, showCmd, sendCmd, summaryCmd, payCmd, versionCmd} {
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
NIF de emitente/cliente e envio por email. Inclui relatório anual com breakdown
por cliente, registo de pagamentos e exportação em PDF e CSV.

Comandos disponíveis:
  generate   Gerar um PDF de fatura
  list       Listar faturas emitidas (com estado de pagamento)
  show       Mostrar detalhes de uma fatura
  send       Enviar fatura por email
  pay        Marcar fatura como paga
  summary    Resumo anual: tabela mensal, clientes, pendentes, projeção, CSV, PDF
  version    Mostrar a versão instalada

Exemplos:
  fatura generate --from "Empresa, Lda." --to "Cliente, S.A." \
    --item "Consultoria" --rate 500 --iva 0.23 \
    --seller-vat-id PT501234567 --buyer-vat-id PT509876543

  fatura generate --client empresa-xpto --item "Manutenção" --rate 200 --iva 0.23

  fatura list
  fatura show INV-2026-001
  fatura pay INV-2026-001
  fatura summary
  fatura summary --pdf relatorio-2026.pdf --compare 2025
  fatura send --to cliente@exemplo.pt --pdf fatura.pdf

Configuração guardada em ~/.fatura/ (histórico, clientes, modelos, SMTP, CSV anual).`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Gerar uma fatura em PDF",
	Long: `Gera um PDF de fatura e guarda-o automaticamente em:
  ~/.fatura/history/<cliente>/<ano>/<mês>/<id>-<cliente>.pdf

O número de fatura é atribuído automaticamente no formato INV-YYYY-NNN
(ex: INV-2026-001). O contador anual é guardado em ~/.fatura/counter.json.

Após cada fatura real (não rascunho), o relatório anual CSV é actualizado:
  ~/.fatura/relatorio-YYYY-MOEDA.csv

Com --draft, é gerado um rascunho com marca de água "RASCUNHO" sem consumir
um número do contador e sem actualizar o histórico nem o CSV.

Configuração de cliente:
  Guarde dados recorrentes em ~/.fatura/clients/<nome>.yaml e carregue-os
  com --client <nome>. Os flags da linha de comandos têm sempre precedência.

Isenção de IVA:
  Use --exemption <código> com um código AT (ex: M07, M09). O motivo e a
  referência legal correspondentes são preenchidos automaticamente.

ATCUD:
  Obtenha o código de validação no portal AT e indique-o com --atcud-code.
  O ATCUD aparece na fatura como <código>-<número sequencial>. Se omitido,
  a secção ATCUD não é apresentada no documento.

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

		// Normalise dates: accept YYYY-MM-DD and DD/MM/YYYY in addition to the
		// canonical "Jan 02, 2006" format used everywhere internally.
		var errDate error
		if invoice.Date, errDate = normalizeDate(invoice.Date); errDate != nil {
			return fmt.Errorf("--date: %w", errDate)
		}
		if invoice.Due, errDate = normalizeDate(invoice.Due); errDate != nil {
			return fmt.Errorf("--due: %w", errDate)
		}

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

		if err := pdf.AddTTFFontData("Sans", plexSansFont); err != nil {
			return err
		}
		if err := pdf.AddTTFFontData("Sans-B", plexSansBoldFont); err != nil {
			return err
		}
		if err := pdf.AddTTFFontData("Mono", plexMonoFont); err != nil {
			return err
		}
		if err := pdf.AddTTFFontData("Mono-B", plexMonoBoldFont); err != nil {
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

		taxAmount := subtotal * invoice.Tax
		total := subtotal + taxAmount - subtotal*invoice.Withholding - subtotal*invoice.Discount
		if !draft {
			_ = saveToHistory(invoice, output, subtotal, taxAmount, total, false)
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
	Long: `Lista todas as faturas guardadas no histórico (~/.fatura/history.json).

Mostra ID, data, cliente, total, estado de pagamento e caminho do PDF.
A coluna ESTADO indica "pago", "pendente" ou "—" (rascunho).
Os rascunhos são assinalados com [rascunho] no ID.

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
		fmt.Fprintln(w, "ID\tDATA\tCLIENTE\tTOTAL\tESTADO\tFICHEIRO")
		for _, r := range records {
			label := r.Id
			if r.Draft {
				label += " [rascunho]"
			}
			estado := "pendente"
			switch {
			case r.Draft:
				estado = "—"
			case r.PaidAt != "":
				estado = "pago"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s%.2f\t%s\t%s\n",
				label, r.Date, truncate(r.To, 30),
				currencySymbol(r.Currency), r.Total, estado, r.PDF)
		}
		return w.Flush()
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Mostrar detalhes de uma fatura",
	Long: `Mostra os detalhes completos de uma fatura a partir do seu ID, incluindo
o estado de pagamento (pendente / pago em YYYY-MM-DD HH:MM).

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
				switch {
				case r.Draft:
					fmt.Println("Tipo:       Rascunho")
				case r.PaidAt != "":
					fmt.Printf("Pago em:    %s\n", r.PaidAt)
				default:
					fmt.Println("Estado:     Pendente")
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

// normalizeDate accepts "Jan 02, 2006", "2006-01-02", or "02/01/2006" and
// always returns the canonical "Jan 02, 2006" form used internally.
func normalizeDate(s string) (string, error) {
	for _, layout := range []string{"Jan 02, 2006", "2006-01-02", "02/01/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			year := t.Year()
			if year < 2000 || year > time.Now().Year()+5 {
				return "", fmt.Errorf("ano improvável: %d — verifique a data %q (pretendia %s)",
					year, s, suggestYear(year))
			}
			return t.Format("Jan 02, 2006"), nil
		}
	}
	return "", fmt.Errorf("formato de data inválido: use YYYY-MM-DD, DD/MM/YYYY ou \"Jan 02, 2006\"")
}

// suggestYear proposes the most likely correction for a mistyped year.
// e.g. 1015 → "2015?", 1025 → "2025?" (swap leading 1 → 2).
func suggestYear(y int) string {
	s := fmt.Sprintf("%d", y)
	if len(s) == 4 && s[0] == '1' {
		return "2" + s[1:] + "?"
	}
	return fmt.Sprintf("%d?", time.Now().Year())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

var summaryCmd = &cobra.Command{
	Use:   "summary [ano]",
	Short: "Resumo anual de faturação",
	Long: `Apresenta o resumo mensal de faturação para o ano indicado (ou o ano actual).

Mostra subtotal faturado, IVA, retenção na fonte e total por mês, com totais
anuais, breakdown por cliente, recebíveis pendentes e projeção anual.
O CSV é actualizado automaticamente após cada fatura:
  ~/.fatura/relatorio-YYYY-MOEDA.csv

Opções:
  --pdf <ficheiro>    Gerar relatório em PDF
  --compare <ano>     Comparar com outro ano

Exemplos:
  fatura summary
  fatura summary 2025
  fatura summary --pdf relatorio-2026.pdf
  fatura summary --compare 2025`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		year := time.Now().Year()
		if len(args) == 1 {
			y, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("ano inválido: %q", args[0])
			}
			year = y
		}
		pdfOut, _ := cmd.Flags().GetString("pdf")
		compareYear, _ := cmd.Flags().GetInt("compare")

		records, err := loadHistory()
		if err != nil {
			return err
		}

		// Collect currencies present in the requested year.
		currencySet := map[string]struct{}{}
		for _, r := range records {
			if r.Draft {
				continue
			}
			t, err := time.Parse("Jan 02, 2006", r.Date)
			if err != nil || t.Year() != year {
				continue
			}
			cur := r.Currency
			if cur == "" {
				cur = "EUR"
			}
			currencySet[cur] = struct{}{}
		}

		if len(currencySet) == 0 {
			fmt.Printf("Nenhuma fatura registada para %d.\n", year)
			return nil
		}

		for currency := range currencySet {
			months, clients, unpaid, totSub, totTax, totWith, totTotal := aggregateYear(records, year, currency)
			sym := currencySymbol(currency)

			// ── Monthly table ────────────────────────────────────────────────
			fmt.Printf("Relatório anual — %d (%s)\n\n", year, currency)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintf(w, "Mês\tFaturado\tIVA\tRetenção IRS\tTotal\n")
			for i, name := range ptMonths {
				fmt.Fprintf(w, "%s\t%s%.2f\t%s%.2f\t%s%.2f\t%s%.2f\n",
					name,
					sym, months[i].subtotal,
					sym, months[i].taxAmount,
					sym, months[i].withholding,
					sym, months[i].total)
			}
			fmt.Fprintf(w, "TOTAL\t%s%.2f\t%s%.2f\t%s%.2f\t%s%.2f\n",
				sym, totSub, sym, totTax, sym, totWith, sym, totTotal)
			_ = w.Flush()

			// ── Per-client breakdown ─────────────────────────────────────────
			if len(clients) > 0 {
				fmt.Printf("\nPor cliente:\n")
				wc := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				for _, c := range clients {
					inv := "fatura"
					if c.invoices != 1 {
						inv = "faturas"
					}
					pendente := ""
					if c.unpaid > 0 {
						pendente = fmt.Sprintf("  (pendente %s%.2f)", sym, c.unpaid)
					}
					fmt.Fprintf(wc, "  %s\t%d %s\t%s%.2f%s\n",
						c.name, c.invoices, inv, sym, c.total, pendente)
				}
				_ = wc.Flush()
			}

			// ── Unpaid receivables ───────────────────────────────────────────
			if len(unpaid) > 0 {
				var unpaidTotal float64
				for _, r := range unpaid {
					unpaidTotal += r.Total
				}
				fmt.Printf("\nRecebíveis pendentes: %s%.2f (%d fatura(s))\n",
					sym, unpaidTotal, len(unpaid))
				for _, r := range unpaid {
					client := strings.TrimSpace(strings.Split(strings.ReplaceAll(r.To, `\n`, "\n"), "\n")[0])
					fmt.Printf("  %-16s  %-28s  %s  %s%.2f\n",
						r.Id, truncate(client, 28), r.Date, sym, r.Total)
				}
			}

			// ── Projection ───────────────────────────────────────────────────
			if year == time.Now().Year() {
				monthsElapsed := int(time.Now().Month())
				if totTotal > 0 && monthsElapsed > 0 {
					projected := totTotal / float64(monthsElapsed) * 12
					fmt.Printf("\nProjeção anual (%d meses): %s%.2f\n",
						monthsElapsed, sym, projected)
				}
			}

			// ── YoY comparison ───────────────────────────────────────────────
			if compareYear > 0 {
				var cmpTotal float64
				for _, r := range records {
					if r.Draft {
						continue
					}
					cur := r.Currency
					if cur == "" {
						cur = "EUR"
					}
					t, err := time.Parse("Jan 02, 2006", r.Date)
					if err != nil || t.Year() != compareYear || cur != currency {
						continue
					}
					cmpTotal += r.Total
				}
				if cmpTotal > 0 {
					delta := (totTotal - cmpTotal) / cmpTotal * 100
					sign := "+"
					if delta < 0 {
						sign = ""
					}
					fmt.Printf("\nComparação: %d %s%.2f  vs  %d %s%.2f  (%s%.1f%%)\n",
						year, sym, totTotal, compareYear, sym, cmpTotal, sign, delta)
				}
			}

			// ── CSV path ─────────────────────────────────────────────────────
			if csvPath, err := yearlyCSVPath(year, currency); err == nil {
				fmt.Printf("\nCSV: %s\n", csvPath)
			}

			// ── PDF report ───────────────────────────────────────────────────
			if pdfOut != "" {
				if err := generateAnnualReportPDF(records, year, compareYear, currency, pdfOut); err != nil {
					return fmt.Errorf("erro ao gerar PDF: %w", err)
				}
				fmt.Printf("Relatório PDF: %s\n", pdfOut)
			}
		}
		return nil
	},
}

var payCmd = &cobra.Command{
	Use:   "pay <id>",
	Short: "Marcar fatura como paga",
	Long: `Marca uma fatura como paga, registando a data e hora de pagamento.

Exemplo:
  fatura pay INV-2026-001`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paidAt, err := markAsPaid(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Fatura %s marcada como paga em %s.\n", args[0], paidAt)
		return nil
	},
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
