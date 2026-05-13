# fatura

Gerador de faturas portuguesas em linha de comandos. Gera PDFs conformes com os requisitos da Autoridade Tributária (AT), incluindo IVA, isenções, ATCUD e numeração sequencial automática. Inclui relatório anual com breakdown por cliente, registo de pagamentos e exportação em PDF e CSV.

## Instalação

```sh
go install github.com/lisbonjoker/fatura@latest
```

## Utilização rápida

```bash
fatura generate \
  --from "A Minha Empresa, Lda.\nRua Exemplo, 1\n1000-001 Lisboa\nNIF: 501234567" \
  --to "Empresa Cliente, S.A.\nAv. República, 50\n1050-187 Lisboa" \
  --item "Serviços de consultoria" --quantity 8 --rate 75 \
  --iva 0.23 \
  --seller-vat-id "PT501234567" --buyer-vat-id "PT509876543" \
  --payment-terms "30 dias"

# Ver resumo do ano com breakdown por cliente
fatura summary

# Marcar fatura como paga
fatura pay INV-2026-001

# Gerar relatório anual em PDF com comparação ao ano anterior
fatura summary --pdf relatorio-2026.pdf --compare 2025
```

O PDF é guardado automaticamente em `~/.fatura/history/<cliente>/<ano>/<mês>/<id>-<cliente>.pdf`.

---

## Comandos

### `fatura generate`

Gera um PDF de fatura e guarda-o no histórico. Actualiza automaticamente o CSV anual em `~/.fatura/relatorio-YYYY-MOEDA.csv`.

```
fatura generate [flags]
```

**Flags principais:**

| Flag | Descrição |
|------|-----------|
| `--from` | Nome e morada do emitente |
| `--to` | Nome e morada do destinatário |
| `--from-line` | Linha do emitente (repetível, alternativa a `--from`) |
| `--to-line` | Linha do destinatário (repetível) |
| `--item` / `-i` | Descrição do artigo ou serviço (repetível) |
| `--quantity` / `-q` | Quantidade (repetível, suporta decimais como `0.25`) |
| `--rate` / `-r` | Preço unitário (repetível) |
| `--iva` | Taxa de IVA (ex: `0.23` para 23%) — adicionado ao subtotal |
| `--withholding` | Taxa de retenção na fonte IRS (ex: `0.25` para 25%) — deduzido do total a pagar |
| `--discount` / `-d` | Desconto (ex: `0.10` para 10%) |
| `--currency` / `-c` | Moeda (`EUR`, `USD`, `GBP`; predefinição: `EUR`) |
| `--date` | Data de emissão (predefinição: hoje) |
| `--due` | Data de vencimento (predefinição: 30 dias) |
| `--seller-vat-id` | NIF do fornecedor (ex: `PT501234567`) |
| `--buyer-vat-id` | NIF do cliente |
| `--exemption` | Código de isenção AT (ex: `M07`, `M09`) |
| `--reference` | Referência de encomenda/PO (ex: `PO-2026-001`) |
| `--atcud-code` | Código de validação ATCUD (obtido no portal AT) |
| `--payment-terms` | Condições de pagamento (ex: `30 dias`) |
| `--note` / `-n` | Observações no rodapé |
| `--logo` / `-l` | Caminho para o logótipo (PNG/JPG) |
| `--id` | Número de fatura manual (gerado automaticamente se omitido) |
| `--draft` | Gerar rascunho com marca de água (não incrementa contador) |
| `--client` | Carregar configuração de cliente em `~/.fatura/clients/<nome>.yaml` |
| `--recur` | Guardar como modelo recorrente com este nome |
| `--item-columns` | Colunas da tabela: `date,time,category,qty,rate,amount` |
| `--item-date` | Data por artigo (repetível) |
| `--item-time` | Hora por artigo, ex: `09:00-17:00` (repetível) |
| `--item-category` | Categoria/código por artigo (repetível) |
| `--output` / `-o` | Caminho de saída do PDF (substitui o caminho automático) |
| `--import` | Importar dados de ficheiro `.json` ou `.yaml` |
| `--title` | Título do documento (predefinição: `FATURA`) |

**Numeração automática:** Cada fatura recebe um número sequencial no formato `INV-YYYY-NNN` (ex: `INV-2026-001`). O contador é guardado em `~/.fatura/counter.json` por ano.

**Rascunho:** Com `--draft`, é gerado um PDF com marca de água `RASCUNHO`. O número não é consumido do contador.

---

### `fatura list`

Lista todas as faturas emitidas com ID, data, cliente, total, estado de pagamento e caminho do PDF.

```bash
fatura list
```

A coluna ESTADO mostra `pago`, `pendente` ou `—` (rascunho).

---

### `fatura show <id>`

Mostra os detalhes completos de uma fatura, incluindo o estado de pagamento.

```bash
fatura show INV-2026-001
```

---

### `fatura pay <id>`

Marca uma fatura como paga, registando a data e hora de pagamento em `~/.fatura/history.json`. O estado fica visível em `fatura list` e `fatura show`.

```bash
fatura pay INV-2026-001
# Fatura INV-2026-001 marcada como paga em 2026-05-13 14:30.
```

---

### `fatura summary [ano]`

Apresenta o resumo anual de faturação para o ano indicado (ou o ano corrente). O CSV em `~/.fatura/relatorio-YYYY-MOEDA.csv` é actualizado automaticamente após cada `fatura generate`.

```bash
fatura summary           # ano corrente
fatura summary 2025      # ano específico
```

**Saída:**

- **Tabela mensal** — Faturado (subtotal), IVA cobrado, Retenção IRS e Total líquido para cada mês, com totais anuais.
- **Por cliente** — ranking de clientes por volume faturado, com número de faturas e montante pendente.
- **Recebíveis pendentes** — lista de faturas ainda não marcadas como pagas.
- **Projeção anual** — estimativa do total anual com base no ritmo dos meses decorridos (só no ano corrente).
- **Comparação anual** — variação em percentagem face ao ano indicado em `--compare`.

**Opções:**

| Flag | Descrição |
|------|-----------|
| `--compare <ano>` | Comparar com o ano indicado (ex: `--compare 2025`) |
| `--pdf <ficheiro>` | Gerar relatório em PDF para o ficheiro indicado |

```bash
fatura summary --compare 2025
fatura summary --pdf relatorio-2026.pdf
fatura summary 2025 --pdf relatorio-2025.pdf --compare 2024
```

**Relatório PDF** (`--pdf`):

O PDF utiliza o mesmo estilo Swiss Grid das faturas e inclui:
- Cabeçalho com total anual em destaque
- Tabela mensal completa
- Gráfico de barras por mês (barras laranja; barras cinzentas para o ano de comparação quando `--compare` é utilizado)
- Breakdown por cliente (até 8, ordenados por volume)
- Lista de recebíveis pendentes (até 6)
- Caixa de estatísticas: projeção e variação YoY

---

### `fatura send`

Envia uma fatura por email (requer SMTP configurado).

```bash
fatura send --to cliente@exemplo.pt --pdf caminho/para/fatura.pdf
fatura send --to cliente@exemplo.pt --pdf fatura.pdf --subject "Fatura INV-2026-001"
```

---

## Relatório anual em CSV

O ficheiro `~/.fatura/relatorio-YYYY-MOEDA.csv` é criado ou actualizado automaticamente após cada `fatura generate` (faturas reais, não rascunhos). Abre directamente em Excel, Numbers ou qualquer folha de cálculo.

```
Mês,Faturado,IVA,Retenção IRS,Total
Janeiro,0.00,0.00,0.00,0.00
Fevereiro,1000.00,230.00,250.00,980.00
...
Dezembro,0.00,0.00,0.00,0.00
TOTAL,1000.00,230.00,250.00,980.00
```

Existindo faturas em múltiplas moedas, é gerado um CSV por moeda (ex: `relatorio-2026-EUR.csv`, `relatorio-2026-USD.csv`).

---

## Configuração de clientes

Guarde as configurações recorrentes de um cliente em `~/.fatura/clients/<nome>.yaml`:

```yaml
# ~/.fatura/clients/empresa-xpto.yaml
to: "Empresa XPTO, S.A.\nRua das Flores, 10\n4000-001 Porto"
buyer_vat_id: "PT509876543"
payment_terms: "30 dias"
currency: EUR
```

Utilizar o cliente guardado:

```bash
fatura generate --client empresa-xpto \
  --item "Desenvolvimento de software" --quantity 20 --rate 90 \
  --iva 0.23
```

---

## Modelos recorrentes

Guarde uma fatura como modelo para reutilização futura:

```bash
fatura generate --client empresa-xpto \
  --item "Manutenção mensal" --rate 500 \
  --iva 0.23 \
  --recur manutencao-mensal
```

Reutilizar o modelo no mês seguinte:

```bash
fatura generate --import ~/.fatura/recurring/manutencao-mensal.yaml
```

---

## Isenção de IVA

Quando o IVA é zero, utilize `--exemption` com o código AT correspondente:

```bash
fatura generate \
  --item "Formação profissional" --rate 1200 \
  --exemption M09
```

Códigos mais comuns:

| Código | Motivo | Referência Legal |
|--------|--------|------------------|
| M01 | Artigo 16.º n.º 6 do CIVA | Art. 16.º n.º 6 CIVA |
| M02 | Artigo 6.º do Decreto‑Lei n.º 198/90 | DL n.º 198/90 |
| M04 | Isento — Artigo 13.º do CIVA | Art. 13.º CIVA |
| M05 | Isento — Artigo 14.º do CIVA | Art. 14.º CIVA |
| M06 | Isento — Artigo 15.º do CIVA | Art. 15.º CIVA |
| M07 | Isento — Artigo 9.º do CIVA | Art. 9.º CIVA |
| M09 | IVA — Não confere direito a dedução | Art. 62.º alínea b) CIVA |
| M10 | IVA — Regime de isenção | Art. 57.º CIVA |
| M16 | Isento — Artigo 14.º do RITI | Art. 14.º RITI |
| M19 | Outras isenções | Isenções definitivas |
| M20 | IVA — Regime forfetário | Art. 59.º-D n.º 2 CIVA |
| M99 | Não sujeito; não tributado | — |

Para um motivo personalizado em vez de um código AT, utilize `--exemption-reason` e `--legal-reference`.

---

## Retenção na fonte IRS

A retenção na fonte é deduzida do montante que o cliente transfere, ao contrário do IVA que é adicionado. O cálculo aplica-se sempre sobre o subtotal, nunca sobre o IVA.

```
Subtotal          €1.000,00
IVA (23%)        +€  230,00
Retenção IRS (25%) -€ 250,00
─────────────────────────────
Total a pagar      €  980,00
```

A taxa padrão para a maioria dos trabalhadores independentes a faturar a empresas portuguesas é de **25%**. Pode ser omitida ou definida como `0` para:
- Clientes internacionais (fora de Portugal)
- Freelancers com rendimento anual abaixo de €14.500 (isenção de retenção)

```bash
fatura generate \
  --item "Desenvolvimento web" --quantity 10 --rate 100 \
  --iva 0.23 --withholding 0.25 \
  --seller-vat-id PT501234567 --buyer-vat-id PT509876543
```

---

## ATCUD

O ATCUD é obrigatório para faturas emitidas por software certificado pela AT. Obtenha o código de validação no portal AT e indique-o com `--atcud-code`:

```bash
fatura generate \
  --atcud-code "CSDF7T5V" \
  --item "Consultoria" --rate 500 \
  --iva 0.23
```

O ATCUD será apresentado na fatura como `CSDF7T5V-001` (código de validação + número sequencial). Quando não é fornecido, a secção ATCUD não aparece no documento.

---

## Configuração SMTP (envio de email)

Crie o ficheiro `~/.fatura/config.yaml`:

```yaml
smtp:
  host: smtp.exemplo.pt
  port: 587        # 587 para STARTTLS, 465 para TLS implícito
  user: utilizador@exemplo.pt
  password: palavra-passe
  from: A Minha Empresa <faturacao@exemplo.pt>
```

Enviar uma fatura:

```bash
fatura send \
  --to cliente@exemplo.pt \
  --pdf ~/.fatura/history/empresa-cliente/2026/05/INV-2026-001-empresa-cliente.pdf \
  --subject "Fatura INV-2026-001 — Maio 2026"
```

---

## Estrutura de ficheiros

```
~/.fatura/
├── config.yaml                    # Configuração SMTP e global
├── counter.json                   # Contadores de numeração por ano
├── history.json                   # Histórico de faturas (totais, estado de pagamento)
├── relatorio-2026-EUR.csv         # Relatório anual CSV (criado automaticamente)
├── clients/
│   └── empresa-xpto.yaml          # Configuração por cliente
├── recurring/
│   └── manutencao-mensal.yaml     # Modelos recorrentes
└── history/
    └── empresa-xpto/
        └── 2026/
            └── 05/
                └── INV-2026-001-empresa-xpto.pdf
```

---

## Importar de ficheiro

Pode definir todos os campos num ficheiro `.json` ou `.yaml` e importar com `--import`. Os flags da linha de comandos têm precedência sobre os valores do ficheiro.

```yaml
# fatura.yaml
from: "A Minha Empresa, Lda.\nRua Exemplo, 1\n1000-001 Lisboa"
to: "Empresa Cliente, S.A."
seller_vat_id: "PT501234567"
buyer_vat_id: "PT509876543"
items:
  - "Desenvolvimento de aplicação web"
  - "Reuniões de acompanhamento"
quantities:
  - 40
  - 4
rates:
  - 90
  - 90
tax: 0.23
currency: EUR
payment_terms: "30 dias"
item_columns: "qty,rate,amount"
```

```bash
fatura generate --import fatura.yaml --note "Referente ao mês de maio de 2026."
```

---

## Variáveis de ambiente

Todas as flags também podem ser definidas por variáveis de ambiente com o prefixo `FATURA_`:

```bash
export FATURA_FROM="A Minha Empresa, Lda."
export FATURA_SELLER_VAT_ID="PT501234567"
export FATURA_IVA=0.23
```

---

## Licença

[MIT](./LICENSE)

---

## Comandos

### `fatura generate`

Gera um PDF de fatura e guarda-o no histórico.

```
fatura generate [flags]
```

**Flags principais:**

| Flag | Descrição |
|------|-----------|
| `--from` | Nome e morada do emitente |
| `--to` | Nome e morada do destinatário |
| `--from-line` | Linha do emitente (repetível, alternativa a `--from`) |
| `--to-line` | Linha do destinatário (repetível) |
| `--item` / `-i` | Descrição do artigo ou serviço (repetível) |
| `--quantity` / `-q` | Quantidade (repetível, suporta decimais como `0.25`) |
| `--rate` / `-r` | Preço unitário (repetível) |
| `--iva` | Taxa de IVA (ex: `0.23` para 23%) — adicionado ao subtotal |
| `--withholding` | Taxa de retenção na fonte IRS (ex: `0.25` para 25%) — deduzido do total a pagar |
| `--discount` / `-d` | Desconto (ex: `0.10` para 10%) |
| `--currency` / `-c` | Moeda (`EUR`, `USD`, `GBP`; predefinição: `EUR`) |
| `--date` | Data de emissão (predefinição: hoje) |
| `--due` | Data de vencimento (predefinição: 30 dias) |
| `--seller-vat-id` | NIF do fornecedor (ex: `PT501234567`) |
| `--buyer-vat-id` | NIF do cliente |
| `--exemption` | Código de isenção AT (ex: `M07`, `M09`) |
| `--reference` | Referência de encomenda/PO (ex: `PO-2026-001`) |
| `--atcud-code` | Código de validação ATCUD (obtido no portal AT) |
| `--payment-terms` | Condições de pagamento (ex: `30 dias`) |
| `--note` / `-n` | Observações no rodapé |
| `--logo` / `-l` | Caminho para o logótipo (PNG/JPG) |
| `--id` | Número de fatura manual (gerado automaticamente se omitido) |
| `--draft` | Gerar rascunho com marca de água (não incrementa contador) |
| `--client` | Carregar configuração de cliente em `~/.fatura/clients/<nome>.yaml` |
| `--recur` | Guardar como modelo recorrente com este nome |
| `--item-columns` | Colunas da tabela: `date,time,category,qty,rate,amount` |
| `--item-date` | Data por artigo (repetível) |
| `--item-time` | Hora por artigo, ex: `09:00-17:00` (repetível) |
| `--item-category` | Categoria/código por artigo (repetível) |
| `--output` / `-o` | Caminho de saída do PDF (substitui o caminho automático) |
| `--import` | Importar dados de ficheiro `.json` ou `.yaml` |
| `--title` | Título do documento (predefinição: `FATURA`) |

**Numeração automática:** Cada fatura recebe um número sequencial no formato `INV-YYYY-NNN` (ex: `INV-2026-001`). O contador é guardado em `~/.fatura/counter.json` por ano.

**Rascunho:** Com `--draft`, é gerado um PDF com marca de água `RASCUNHO`. O número não é consumido do contador.

---

### `fatura list`

Lista todas as faturas emitidas.

```bash
fatura list
```

Mostra ID, data, cliente, total e caminho do PDF para cada fatura no histórico.

---

### `fatura show <id>`

Mostra os detalhes de uma fatura pelo seu ID.

```bash
fatura show INV-2026-001
```

---

### `fatura send`

Envia uma fatura por email (requer SMTP configurado).

```bash
fatura send --to cliente@exemplo.pt --pdf caminho/para/fatura.pdf
fatura send --to cliente@exemplo.pt --pdf fatura.pdf --subject "Fatura INV-2026-001"
```

---

## Configuração de clientes

Guarde as configurações recorrentes de um cliente em `~/.fatura/clients/<nome>.yaml`:

```yaml
# ~/.fatura/clients/empresa-xpto.yaml
to: "Empresa XPTO, S.A.\nRua das Flores, 10\n4000-001 Porto"
buyer_vat_id: "PT509876543"
payment_terms: "30 dias"
currency: EUR
```

Utilizar o cliente guardado:

```bash
fatura generate --client empresa-xpto \
  --item "Desenvolvimento de software" --quantity 20 --rate 90 \
  --iva 0.23
```

---

## Modelos recorrentes

Guarde uma fatura como modelo para reutilização futura:

```bash
fatura generate --client empresa-xpto \
  --item "Manutenção mensal" --rate 500 \
  --iva 0.23 \
  --recur manutencao-mensal
```

Reutilizar o modelo no mês seguinte:

```bash
fatura generate --import ~/.fatura/recurring/manutencao-mensal.yaml
```

---

## Isenção de IVA

Quando o IVA é zero, utilize `--exemption` com o código AT correspondente:

```bash
fatura generate \
  --item "Formação profissional" --rate 1200 \
  --exemption M09
```

Códigos mais comuns:

| Código | Motivo | Referência Legal |
|--------|--------|------------------|
| M01 | Artigo 16.º n.º 6 do CIVA | Art. 16.º n.º 6 CIVA |
| M02 | Artigo 6.º do Decreto‑Lei n.º 198/90 | DL n.º 198/90 |
| M04 | Isento — Artigo 13.º do CIVA | Art. 13.º CIVA |
| M05 | Isento — Artigo 14.º do CIVA | Art. 14.º CIVA |
| M06 | Isento — Artigo 15.º do CIVA | Art. 15.º CIVA |
| M07 | Isento — Artigo 9.º do CIVA | Art. 9.º CIVA |
| M09 | IVA — Não confere direito a dedução | Art. 62.º alínea b) CIVA |
| M10 | IVA — Regime de isenção | Art. 57.º CIVA |
| M16 | Isento — Artigo 14.º do RITI | Art. 14.º RITI |
| M19 | Outras isenções | Isenções definitivas |
| M20 | IVA — Regime forfetário | Art. 59.º-D n.º 2 CIVA |
| M99 | Não sujeito; não tributado | — |

Para um motivo personalizado em vez de um código AT, utilize `--exemption-reason` e `--legal-reference`.

---

## Retenção na fonte IRS

A retenção na fonte é deduzida do montante que o cliente transfere, ao contrário do IVA que é adicionado. O cálculo aplica-se sempre sobre o subtotal, nunca sobre o IVA.

```
Subtotal          €1.000,00
IVA (23%)        +€  230,00
Retenção IRS (25%) -€ 250,00
─────────────────────────────
Total a pagar      €  980,00
```

A taxa padrão para a maioria dos trabalhadores independentes a faturar a empresas portuguesas é de **25%**. Pode ser omitida ou definida como `0` para:
- Clientes internacionais (fora de Portugal)
- Freelancers com rendimento anual abaixo de €14.500 (isenção de retenção)

```bash
fatura generate \
  --item "Desenvolvimento web" --quantity 10 --rate 100 \
  --iva 0.23 --withholding 0.25 \
  --seller-vat-id PT501234567 --buyer-vat-id PT509876543
```

---

## ATCUD

O ATCUD é obrigatório para faturas emitidas por software certificado pela AT. Obtenha o código de validação no portal AT e indique-o com `--atcud-code`:

```bash
fatura generate \
  --atcud-code "CSDF7T5V" \
  --item "Consultoria" --rate 500 \
  --iva 0.23
```

O ATCUD será apresentado na fatura como `CSDF7T5V-001` (código de validação + número sequencial).

---

## Configuração SMTP (envio de email)

Crie o ficheiro `~/.fatura/config.yaml`:

```yaml
smtp:
  host: smtp.exemplo.pt
  port: 587        # 587 para STARTTLS, 465 para TLS implícito
  user: utilizador@exemplo.pt
  password: palavra-passe
  from: A Minha Empresa <faturacao@exemplo.pt>
```

Enviar uma fatura:

```bash
fatura send \
  --to cliente@exemplo.pt \
  --pdf ~/.fatura/history/empresa-cliente/2026/05/INV-2026-001-empresa-cliente.pdf \
  --subject "Fatura INV-2026-001 — Maio 2026"
```

---

## Estrutura de ficheiros

```
~/.fatura/
├── config.yaml              # Configuração SMTP e global
├── counter.json             # Contadores de numeração por ano
├── history.json             # Histórico de faturas emitidas
├── clients/
│   └── empresa-xpto.yaml   # Configuração por cliente
├── recurring/
│   └── manutencao-mensal.yaml  # Modelos recorrentes
└── history/
    └── empresa-xpto/
        └── 2026/
            └── 05/
                └── INV-2026-001-empresa-xpto.pdf
```

---

## Importar de ficheiro

Pode definir todos os campos num ficheiro `.json` ou `.yaml` e importar com `--import`. Os flags da linha de comandos têm precedência sobre os valores do ficheiro.

```yaml
# fatura.yaml
from: "A Minha Empresa, Lda.\nRua Exemplo, 1\n1000-001 Lisboa"
to: "Empresa Cliente, S.A."
seller_vat_id: "PT501234567"
buyer_vat_id: "PT509876543"
items:
  - "Desenvolvimento de aplicação web"
  - "Reuniões de acompanhamento"
quantities:
  - 40
  - 4
rates:
  - 90
  - 90
tax: 0.23
currency: EUR
payment_terms: "30 dias"
item_columns: "qty,rate,amount"
```

```bash
fatura generate --import fatura.yaml --note "Referente ao mês de maio de 2026."
```

---

## Variáveis de ambiente

Todas as flags também podem ser definidas por variáveis de ambiente com o prefixo `FATURA_`:

```bash
export FATURA_FROM="A Minha Empresa, Lda."
export FATURA_SELLER_VAT_ID="PT501234567"
export FATURA_IVA=0.23
```

---

## Licença

[MIT](./LICENSE)
