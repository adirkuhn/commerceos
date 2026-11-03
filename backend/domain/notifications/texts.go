package notifications

import (
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/textrender"
)

// Types lists every notification type, each translated in typeTexts.
var Types = []Type{
	TypeImportStarted, TypeImportCompleted, TypeImportFailed,
	TypeDocumentParsingCompleted, TypeDocumentParsingFailed, TypePurchaseOrderNeedsReview,
	TypeReceivingParsingCompleted, TypeReceivingParsingFailed, TypeReceivingDiscrepanciesFound,
	TypeBolNotForSale, TypeStockCloseToExpiry,
	TypeChannelSyncFailing, TypeChannelSyncRecovered,
	TypeDeadLettersWaiting, TypeDeadLettersStale,
}

// Text is a notification's title and message in one language.
type Text struct {
	Title   string
	Message string
}

// TypeText is how one type reads in one language. Tokens are {{param}}
// from the notification's Params. MessageOne replaces Message when the
// "count" param is 1. Parts are appended to the message, in order, only
// when their param is set (e.g. ", 3 orders" only when orders were
// imported). The admin's i18next files (frontend notifications.json,
// types.<type>.title / message / message_one / parts.<param>) mirror these
// exactly -- see TestTypeTexts_matchTheAdminTranslations.
type TypeText struct {
	Title      string
	Message    string
	MessageOne string
	Parts      []TextPart
}

type TextPart struct {
	Param string
	Text  string
}

// Render is t's text in lang (English when unset or missing) with params
// filled in; ok is false for a type without translations, whose stored
// text callers then show as-is.
func (t Type) Render(lang locale.Locale, params map[string]any) (Text, bool) {
	byLang, ok := typeTexts[t]
	if !ok {
		return Text{}, false
	}
	tt, ok := byLang[lang.OrDefault()]
	if !ok {
		tt = byLang[locale.English]
	}
	message := tt.Message
	if tt.MessageOne != "" && isOne(params["count"]) {
		message = tt.MessageOne
	}
	for _, p := range tt.Parts {
		if v, set := params[p.Param]; set && v != nil && v != "" {
			message += p.Text
		}
	}
	return Text{Title: fill(tt.Title, params), Message: fill(message, params)}, true
}

// TypeTexts exposes the translations, e.g. for the admin parity test.
func TypeTexts() map[Type]map[locale.Locale]TypeText { return typeTexts }

func fill(text string, params map[string]any) string {
	return textrender.Render(text, params, func(s string) string { return s })
}

func isOne(v any) bool {
	switch n := v.(type) {
	case int:
		return n == 1
	case int64:
		return n == 1
	case float64:
		return n == 1
	default:
		return false
	}
}

// EmailTexts is the fixed wording around a notification email.
type EmailTexts struct {
	SubjectPrefix string
	OpenLink      string
	Footer        string
}

var emailTexts = map[locale.Locale]EmailTexts{
	locale.English: {
		SubjectPrefix: "CommerceOS: ", OpenLink: "Open in CommerceOS",
		Footer: "You get this because of your notification settings in CommerceOS (Settings, Notifications).",
	},
	locale.Portuguese: {
		SubjectPrefix: "CommerceOS: ", OpenLink: "Abrir no CommerceOS",
		Footer: "Você recebe isto por causa das suas configurações de notificação no CommerceOS (Configurações, Notificações).",
	},
	locale.Dutch: {
		SubjectPrefix: "CommerceOS: ", OpenLink: "Openen in CommerceOS",
		Footer: "Je ontvangt dit vanwege je meldingsinstellingen in CommerceOS (Instellingen, Meldingen).",
	},
}

// EmailTextsFor is the email wording in lang (English when unset).
func EmailTextsFor(lang locale.Locale) EmailTexts {
	if e, ok := emailTexts[lang.OrDefault()]; ok {
		return e
	}
	return emailTexts[locale.English]
}

var typeTexts = map[Type]map[locale.Locale]TypeText{
	TypeImportStarted: {
		locale.English:    {Title: "Import started", Message: "Importing WooCommerce products…"},
		locale.Portuguese: {Title: "Importação iniciada", Message: "Importando produtos do WooCommerce…"},
		locale.Dutch:      {Title: "Import gestart", Message: "WooCommerce-producten worden geïmporteerd…"},
	},
	TypeImportCompleted: {
		locale.English: {Title: "Import completed", Message: "Imported {{products}} products ({{already_linked}} already linked, {{skipped}} skipped)",
			Parts: []TextPart{{"customers", ", {{customers}} customers"}, {"orders", ", {{orders}} orders"}}},
		locale.Portuguese: {Title: "Importação concluída", Message: "{{products}} produtos importados ({{already_linked}} já vinculados, {{skipped}} ignorados)",
			Parts: []TextPart{{"customers", ", {{customers}} clientes"}, {"orders", ", {{orders}} pedidos"}}},
		locale.Dutch: {Title: "Import voltooid", Message: "{{products}} producten geïmporteerd ({{already_linked}} al gekoppeld, {{skipped}} overgeslagen)",
			Parts: []TextPart{{"customers", ", {{customers}} klanten"}, {"orders", ", {{orders}} orders"}}},
	},
	TypeImportFailed: {
		locale.English:    {Title: "Import failed", Message: "{{error}}"},
		locale.Portuguese: {Title: "Falha na importação", Message: "{{error}}"},
		locale.Dutch:      {Title: "Import mislukt", Message: "{{error}}"},
	},
	TypeDocumentParsingCompleted: {
		locale.English:    {Title: "Document parsed", Message: "All line items were matched to products automatically."},
		locale.Portuguese: {Title: "Documento lido", Message: "Todos os itens foram vinculados a produtos automaticamente."},
		locale.Dutch:      {Title: "Document gelezen", Message: "Alle regels zijn automatisch aan producten gekoppeld."},
	},
	TypeDocumentParsingFailed: {
		locale.English:    {Title: "Document parsing failed", Message: "{{error}}"},
		locale.Portuguese: {Title: "Falha ao ler o documento", Message: "{{error}}"},
		locale.Dutch:      {Title: "Document lezen mislukt", Message: "{{error}}"},
	},
	TypePurchaseOrderNeedsReview: {
		locale.English:    {Title: "Purchase order needs review", Message: "Some line items couldn't be matched to a product automatically — review and link them."},
		locale.Portuguese: {Title: "Pedido de compra precisa de revisão", Message: "Alguns itens não puderam ser vinculados a um produto automaticamente — revise e vincule-os."},
		locale.Dutch:      {Title: "Inkooporder moet worden nagekeken", Message: "Sommige regels konden niet automatisch aan een product worden gekoppeld — controleer en koppel ze."},
	},
	TypeReceivingParsingCompleted: {
		locale.English:    {Title: "Invoice parsed", Message: "The invoice matched the purchase order exactly."},
		locale.Portuguese: {Title: "Fatura lida", Message: "A fatura corresponde exatamente ao pedido de compra."},
		locale.Dutch:      {Title: "Factuur gelezen", Message: "De factuur komt precies overeen met de inkooporder."},
	},
	TypeReceivingParsingFailed: {
		locale.English:    {Title: "Invoice parsing failed", Message: "{{error}}"},
		locale.Portuguese: {Title: "Falha ao ler a fatura", Message: "{{error}}"},
		locale.Dutch:      {Title: "Factuur lezen mislukt", Message: "{{error}}"},
	},
	TypeReceivingDiscrepanciesFound: {
		locale.English: {Title: "Receiving discrepancies found",
			Message:    "The invoice didn't fully match the purchase order ({{count}} discrepancies) — review and reconcile.",
			MessageOne: "The invoice didn't fully match the purchase order (1 discrepancy) — review and reconcile."},
		locale.Portuguese: {Title: "Divergências no recebimento",
			Message:    "A fatura não corresponde totalmente ao pedido de compra ({{count}} divergências) — revise e concilie.",
			MessageOne: "A fatura não corresponde totalmente ao pedido de compra (1 divergência) — revise e concilie."},
		locale.Dutch: {Title: "Verschillen bij ontvangst gevonden",
			Message:    "De factuur komt niet helemaal overeen met de inkooporder ({{count}} verschillen) — controleer en stem af.",
			MessageOne: "De factuur komt niet helemaal overeen met de inkooporder (1 verschil) — controleer en stem af."},
	},
	TypeBolNotForSale: {
		locale.English:    {Title: `Bol: "{{product}}" is not for sale`, Message: "{{reason}}"},
		locale.Portuguese: {Title: `Bol: "{{product}}" não está à venda`, Message: "{{reason}}"},
		locale.Dutch:      {Title: `Bol: "{{product}}" is niet te koop`, Message: "{{reason}}"},
	},
	TypeChannelSyncFailing: {
		locale.English:    {Title: "Sync with {{store}} keeps failing", Message: "Orders, stock and statuses are not syncing. Last error: {{error}}"},
		locale.Portuguese: {Title: "A sincronização com {{store}} continua falhando", Message: "Pedidos, estoque e status não estão sincronizando. Último erro: {{error}}"},
		locale.Dutch:      {Title: "Synchronisatie met {{store}} blijft mislukken", Message: "Bestellingen, voorraad en statussen worden niet gesynchroniseerd. Laatste fout: {{error}}"},
	},
	TypeChannelSyncRecovered: {
		locale.English:    {Title: "Sync with {{store}} works again", Message: "Automatic sync has recovered."},
		locale.Portuguese: {Title: "A sincronização com {{store}} voltou a funcionar", Message: "A sincronização automática se recuperou."},
		locale.Dutch:      {Title: "Synchronisatie met {{store}} werkt weer", Message: "De automatische synchronisatie is hersteld."},
	},
	TypeDeadLettersWaiting: {
		locale.English: {Title: "Some automatic actions failed",
			Message:    "{{count}} events could not be processed and are waiting for a retry by support.",
			MessageOne: "1 event could not be processed and is waiting for a retry by support."},
		locale.Portuguese: {Title: "Algumas ações automáticas falharam",
			Message:    "{{count}} eventos não puderam ser processados e aguardam uma nova tentativa do suporte.",
			MessageOne: "1 evento não pôde ser processado e aguarda uma nova tentativa do suporte."},
		locale.Dutch: {Title: "Sommige automatische acties zijn mislukt",
			Message:    "{{count}} events konden niet worden verwerkt en wachten op een nieuwe poging door support.",
			MessageOne: "1 event kon niet worden verwerkt en wacht op een nieuwe poging door support."},
	},
	TypeDeadLettersStale: {
		locale.English: {Title: "Failed automatic actions are still waiting",
			Message:    "{{count}} events have not been processed for {{hours}} hours. Contact support if this keeps happening.",
			MessageOne: "1 event has not been processed for {{hours}} hours. Contact support if this keeps happening."},
		locale.Portuguese: {Title: "Ações automáticas com falha ainda aguardam",
			Message:    "{{count}} eventos não foram processados há {{hours}} horas. Fale com o suporte se isso continuar acontecendo.",
			MessageOne: "1 evento não foi processado há {{hours}} horas. Fale com o suporte se isso continuar acontecendo."},
		locale.Dutch: {Title: "Mislukte automatische acties wachten nog steeds",
			Message:    "{{count}} events zijn al {{hours}} uur niet verwerkt. Neem contact op met support als dit blijft gebeuren.",
			MessageOne: "1 event is al {{hours}} uur niet verwerkt. Neem contact op met support als dit blijft gebeuren."},
	},
	TypeStockCloseToExpiry: {
		locale.English: {Title: "Stock close to expiry", Message: "{{count}} products expire within {{window_days}} days",
			MessageOne: "1 product expires within {{window_days}} days"},
		locale.Portuguese: {Title: "Estoque perto do vencimento", Message: "{{count}} produtos vencem em até {{window_days}} dias",
			MessageOne: "1 produto vence em até {{window_days}} dias"},
		locale.Dutch: {Title: "Voorraad bijna over datum", Message: "{{count}} producten verlopen binnen {{window_days}} dagen",
			MessageOne: "1 product verloopt binnen {{window_days}} dagen"},
	},
}
