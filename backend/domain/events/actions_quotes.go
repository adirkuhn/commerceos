package events

// Saved B2B quotes (backend/application/quotes).
var (
	QuoteCreated = defineAction("quotes.quote.created",
		"A B2B quote was saved for the first time. data: number, name, line_count, total_minor_units, currency_code, customer_id.")
	QuoteUpdated = defineAction("quotes.quote.updated",
		"A saved B2B quote was saved over (changes: name, customer_id, discount_basis_points, currency_code, line_count, total_minor_units).")
	QuoteDeleted = defineAction("quotes.quote.deleted",
		"A saved B2B quote was deleted. data: number, name.")
)
