package messagecenter

import (
	"strings"

	"github.com/adirkuhn/commerceos/domain/locale"
)

// orderStatusLabels are customer-facing names of sales order statuses
// (domain/oms SalesOrderStatus values) for {{order_status_label}}.
var orderStatusLabels = map[locale.Locale]map[string]string{
	locale.English: {
		"pending": "Awaiting payment", "processing": "Processing", "on-hold": "On hold", "shipped": "Shipped",
		"completed": "Completed", "cancelled": "Cancelled", "refunded": "Refunded", "failed": "Failed",
		"partially_refunded": "Partially refunded",
	},
	locale.Portuguese: {
		"pending": "Aguardando pagamento", "processing": "Em processamento", "on-hold": "Em espera", "shipped": "Enviado",
		"completed": "Concluído", "cancelled": "Cancelado", "refunded": "Reembolsado", "failed": "Falhou",
		"partially_refunded": "Reembolsado parcialmente",
	},
	locale.Dutch: {
		"pending": "Wacht op betaling", "processing": "In behandeling", "on-hold": "In de wacht", "shipped": "Verzonden",
		"completed": "Afgerond", "cancelled": "Geannuleerd", "refunded": "Terugbetaald", "failed": "Mislukt",
		"partially_refunded": "Deels terugbetaald",
	},
}

// OrderStatusLabel is a sales order status as a customer reads it, in lang
// (English when unset). An unknown status shows itself, capitalized.
func OrderStatusLabel(status string, lang locale.Locale) string {
	labels, ok := orderStatusLabels[lang.OrDefault()]
	if !ok {
		labels = orderStatusLabels[locale.English]
	}
	if label, ok := labels[status]; ok {
		return label
	}
	if status == "" {
		return ""
	}
	return strings.ToUpper(status[:1]) + status[1:]
}
