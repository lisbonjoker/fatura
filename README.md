<img width="1200" alt="Invoice" src="https://github.com/maaslalani/nap/assets/42545625/16dae9d9-390c-49b6-aedd-3f882b17f57b">

# Invoice

Generate invoices from the command line.

## Command Line Interface

```bash
invoice generate --from "Dream, Inc." --to "Imagine, Inc." \
    --item "Rubber Duck" --quantity 2 --rate 25 \
    --vat 0.23 --discount 0.15 \
    --seller-tax-id "501234567" --buyer-tax-id "509876543" \
    --seller-vat-id "PT501234567" --buyer-vat-id "PT509876543" \
    --country-code "PT" \
    --supply-date "Apr 07, 2026" \
    --note "For debugging purposes."
```

<img src="https://vhs.charm.sh/vhs-66CMd4UQuXkuxX9djHUnGX.gif" width="600" />

View the generated PDF at `invoice.pdf`, you can customize the output location
with `--output`.

```bash
open invoice.pdf
```

<img width="574" alt="Example invoice" src="https://github.com/maaslalani/nap/assets/42545625/13153de2-dfa1-41e6-a18e-4d3a5cea5b74">

### Environment

Save repeated information with environment variables:

```bash
export INVOICE_LOGO=/path/to/image.png
export INVOICE_FROM="Dream, Inc."
export INVOICE_TO="Imagine, Inc."
export INVOICE_TAX=0.13
export INVOICE_RATE=25
```

Generate new invoice:

```bash
invoice generate \
    --item "Yellow Rubber Duck" --quantity 5 \
    --item "Special Edition Plaid Rubber Duck" --quantity 1 \
    --note "For debugging purposes." \
    --output duck-invoice.pdf
```

### Configuration File

Or, save repeated information with JSON / YAML:

```json
{
    "logo": "/path/to/image.png",
    "from": "Dream, Inc.",
    "to": "Imagine, Inc.",
    "tax": 0.23,
    "seller_tax_id": "501234567",
    "buyer_tax_id": "509876543",
    "seller_vat_id": "PT501234567",
    "buyer_vat_id": "PT509876543",
    "country_code": "PT",
    "supply_date": "Apr 07, 2026",
    "exemption_reason": "",
    "legal_reference": "",
    "items": ["Yellow Rubber Duck", "Special Edition Plaid Rubber Duck"],
    "quantities": [5, 1],
    "rates": [25, 25],
}
```

Generate new invoice by importing the configuration file:

```bash
invoice generate --import path/to/data.json \
    --output duck-invoice.pdf
```

### Portuguese / EU VAT fields

For Portuguese and EU compliance workflows, include these flags/fields:

* `--seller-tax-id` / `seller_tax_id`
* `--buyer-tax-id` / `buyer_tax_id`
* `--seller-vat-id` / `seller_vat_id`
* `--buyer-vat-id` / `buyer_vat_id`
* `--supply-date` / `supply_date`
* `--country-code` / `country_code`
* `--vat` (alias for `--tax`)
* `--exemption-reason` / `exemption_reason` when VAT is exempt
* `--legal-reference` / `legal_reference`
* `--pt-exemption` shortcut values: `e_learning`, `gambling`, `insurance_financial`

When `country_code` is `PT`, the CLI now validates key Portuguese invoice requirements:
required supplier/recipient details and VAT IDs, VAT exemption reason + legal reference
when VAT is zero, and issuance within 5 working days of `supply_date`.

### Custom Templates

If you would like a custom invoice template for your business or company, please
reach out via:

* [Email](mailto:maas@lalani.dev)
* [Twitter](https://twitter.com/maaslalani)

## Installation

<!--

Use a package manager:

```bash
# macOS
brew install invoice

# Arch
yay -S invoice

# Nix
nix-env -iA nixpkgs.invoice
```

-->

Install with Go:

```sh
go install github.com/maaslalani/invoice@main
```

Or download a binary from the [releases](https://github.com/maaslalani/invoice/releases).

## License

[MIT](https://github.com/maaslalani/invoice/blob/master/LICENSE)

## Feedback

I'd love to hear your feedback on improving `invoice`.

Feel free to reach out via:
* [Email](mailto:maas@lalani.dev)
* [Twitter](https://twitter.com/maaslalani)
* [GitHub issues](https://github.com/maaslalani/invoice/issues/new)

---

<sub><sub>z</sub></sub><sub>z</sub>z
