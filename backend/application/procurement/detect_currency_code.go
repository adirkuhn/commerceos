package procurement

import "regexp"

// currencyCodePattern mirrors the frontend's currency select (see
// Procurement.tsx's CURRENCIES list) -- the only codes worth detecting
// since they're the only ones a human can actually pick at upload time.
var currencyCodePattern = regexp.MustCompile(`\b(EUR|USD|GBP|CAD|AUD)\b`)

// currencySymbolPattern is the fallback for invoices that print a symbol
// instead of the ISO code (confirmed common in real-world samples -- a
// Dutch invoice showing "323.64 €" with the word "EUR" nowhere on the
// page). $ is inherently ambiguous (USD/CAD/AUD all use it); USD is the
// reasonable default absent any other signal, same tradeoff the frontend's
// own currency picker doesn't need to make since a human just clicks one.
var currencySymbolPattern = regexp.MustCompile(`[€$£]`)

var currencySymbolToCode = map[string]string{"€": "EUR", "$": "USD", "£": "GBP"}

// DetectCurrencyCode is the deterministic-first half of document info
// extraction: a supplier document's currency is usually printed as an ISO
// code or a symbol somewhere in the text (unlike a supplier name, which has
// no reliable fixed pattern), so it's cheap and reliable to spot without
// ever reaching the AI path. Returns ok=false when neither appears anywhere
// in the text.
func DetectCurrencyCode(text string) (code string, ok bool) {
	if match := currencyCodePattern.FindStringSubmatch(text); match != nil {
		return match[1], true
	}
	if symbol := currencySymbolPattern.FindString(text); symbol != "" {
		return currencySymbolToCode[symbol], true
	}
	return "", false
}
