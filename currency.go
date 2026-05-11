package main

var currencySymbols = map[string]string{
	"EUR": "€",
	"USD": "$",
	"GBP": "£",
}

func currencySymbol(currency string) string {
	if sym, ok := currencySymbols[currency]; ok {
		return sym
	}
	return currency + " "
}
