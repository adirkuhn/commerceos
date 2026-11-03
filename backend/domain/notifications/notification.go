package notifications

import "time"

// Type is intentionally open-ended (plain string, not a closed enum) —
// notifications are a general concept any feature can produce into. Import
// jobs are the first producer (see application/channels/create_import_job.go
// and cmd/scheduler's job runner); the type just needs to be unique enough
// for the frontend to pick an icon/style by prefix if it wants to.
type Type string

const (
	TypeImportStarted   Type = "import.started"
	TypeImportCompleted Type = "import.completed"
	TypeImportFailed    Type = "import.failed"

	// TypeDocumentParsingCompleted/Failed are fired by cmd/scheduler's
	// DocumentParsingJob runner (see application/procurement) once a
	// supplier PO/invoice document has been parsed and matched.
	TypeDocumentParsingCompleted Type = "procurement.document_parsing.completed"
	TypeDocumentParsingFailed    Type = "procurement.document_parsing.failed"
	// TypePurchaseOrderNeedsReview is a more specific completed-but-needs-
	// attention variant, fired instead of TypeDocumentParsingCompleted when
	// the parsed document left one or more line items unmatched.
	TypePurchaseOrderNeedsReview Type = "procurement.purchase_order.needs_review"

	// TypeReceivingParsingCompleted/Failed are fired by cmd/scheduler's
	// ReceivingParsingJob runner (see application/procurement) once a
	// supplier invoice has been parsed and diffed against its purchase order.
	TypeReceivingParsingCompleted Type = "procurement.receiving_parsing.completed"
	TypeReceivingParsingFailed    Type = "procurement.receiving_parsing.failed"
	// TypeReceivingDiscrepanciesFound is a more specific completed-but-needs-
	// attention variant, fired instead of TypeReceivingParsingCompleted when
	// the invoice diff produced one or more discrepancies to review.
	TypeReceivingDiscrepanciesFound Type = "procurement.receiving.discrepancies_found"

	// TypeBolNotForSale is fired by PushProductToBolUseCase when Bol's own
	// GET .../not-for-sale-reasons check (called after every push) reports
	// the offer still isn't for sale -- see
	// application/channels/push_product_to_bol.go.
	TypeBolNotForSale Type = "bol.offer.not_for_sale"

	// TypeStockCloseToExpiry is the daily stock-close-to-expiry report's
	// bell entry. Its email is a Message Center template + rule (merchant
	// editable, with extra recipients), so see EmailSentByMessageCenter.
	TypeStockCloseToExpiry Type = "inventory.stock.close_to_expiry"

	// TypeChannelSyncFailing/Recovered: a connection's automatic sync kept
	// failing for channels.SyncFailureAlertAfter, and its first success
	// after that (see application/channels.RecordSyncRunUseCase).
	TypeChannelSyncFailing   Type = "channels.sync_schedule.failing"
	TypeChannelSyncRecovered Type = "channels.sync_schedule.recovered"

	// TypeDeadLettersWaiting/Stale: the daily dead-letter check found events
	// of this tenant a consumer gave up on; Stale once the oldest waited
	// deadletter.StaleAfter (see application/deadletters.CheckDeadLettersUseCase).
	TypeDeadLettersWaiting Type = "deadletters.dead_letter.waiting"
	TypeDeadLettersStale   Type = "deadletters.dead_letter.stale"
)

// messageCenterEmailTypes are the types whose email a Message Center rule
// sends, so the generic notification email must not send it again.
var messageCenterEmailTypes = map[Type]bool{TypeStockCloseToExpiry: true}

// EmailSentByMessageCenter reports whether a Message Center rule, not the
// generic notification email, sends this type's email.
func (t Type) EmailSentByMessageCenter() bool { return messageCenterEmailTypes[t] }

type Notification struct {
	ID       string
	TenantID string
	Type     Type
	// Params are the values the type's translated text is filled with
	// (see Type.Render) -- each reader sees it in their own language.
	// Title/Message keep the English text for types without translations
	// and for clients that predate Params.
	Params  map[string]any
	Title   string
	Message string
	// Link is a frontend-relative deep link (e.g. "/products/<id>") the
	// notification navigates to on click -- empty means no link, click
	// just marks it read (every notification before this field existed
	// behaves exactly this way, unchanged).
	Link      string
	Read      bool
	CreatedAt time.Time
}
