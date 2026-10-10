package channels

import (
	"fmt"

	"github.com/adirkuhn/commerceos/domain/locale"
)

var shipmentNoteFormats = map[locale.Locale]string{
	locale.English:    "Shipped via %s, tracking %s",
	locale.Portuguese: "Enviado via %s, rastreio %s",
	locale.Dutch:      "Verzonden met %s, track & trace %s",
}

// ShipmentNote is the order note pushed to a sales channel (e.g. a private
// WooCommerce order note the merchant reads) when an order ships, in lang
// (English when unset). link, when known, is appended.
func ShipmentNote(lang locale.Locale, carrier, barcode, link string) string {
	format, ok := shipmentNoteFormats[lang.OrDefault()]
	if !ok {
		format = shipmentNoteFormats[locale.English]
	}
	note := fmt.Sprintf(format, carrier, barcode)
	if link != "" {
		note += " - " + link
	}
	return note
}
