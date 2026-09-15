package whatsapp

import "strings"

// nameMatchMinScore and nameMatchMinMargin are resolveByName's acceptance
// thresholds -- same values, same algorithm, as
// application/procurement.MatchExtractedLineItemsUseCase's matchByName
// (deliberately duplicated rather than shared -- see
// ResolveSKUFromQueryUseCase's doc comment for why).
const (
	nameMatchMinScore  = 0.6
	nameMatchMinMargin = 0.15
)

// nameMatchScore returns what fraction of productName's own significant
// words are also present in queryTokens -- e.g. catalog "Cha de Limão,
// Gengibre e Mel - Barão de Cotegipe" against a customer query tokenized to
// ["cha", "limao", "gengibre", "mel"] scores high (every query token that
// matters is a product word), while an unrelated product scores low.
// Asymmetric on purpose: a WhatsApp query is usually short ("tem estoque
// do cha de limao?"), while a catalog name often carries an extra brand/
// supplier suffix the query never mentions -- penalizing either side
// equally would punish the normal case this exists for. 0 when productName
// has no significant words.
func nameMatchScore(productName string, queryTokens []string) float64 {
	productTokens := significantTokens(productName)
	if len(productTokens) == 0 {
		return 0
	}
	querySet := make(map[string]struct{}, len(queryTokens))
	for _, t := range queryTokens {
		querySet[t] = struct{}{}
	}
	overlap := 0
	for _, t := range productTokens {
		if _, ok := querySet[t]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(productTokens))
}

// nameMatchStopwords are short connector words common in both Portuguese
// and English product names/queries -- excluded so they don't inflate
// nameMatchScore's overlap count with matches that carry no actual product
// identity (e.g. "e"/"and", "de"/"of", "do"/"tem"/"estoque").
var nameMatchStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "de": {}, "da": {}, "do": {},
	"das": {}, "dos": {}, "em": {}, "para": {}, "com": {}, "por": {},
	"tem": {}, "voce": {}, "vc": {}, "esta": {},
}

// significantTokens lowercases s, strips diacritics, replaces every run of
// non-alphanumeric characters with a single space, then splits into words,
// dropping stopwords and anything under 3 runes.
func significantTokens(s string) []string {
	normalized := stripNameDiacritics(strings.ToLower(s))
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 {
			continue
		}
		if _, stop := nameMatchStopwords[f]; stop {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}

// nameDiacriticsReplacer covers the accented Latin letters that actually
// appear in this catalog's product names (Portuguese, plus common Spanish/
// French overlap) -- a manual table instead of pulling in a
// Unicode-normalization package for what's otherwise a small, fixed
// alphabet.
var nameDiacriticsReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func stripNameDiacritics(s string) string {
	return nameDiacriticsReplacer.Replace(s)
}
