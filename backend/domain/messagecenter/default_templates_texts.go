package messagecenter

import "github.com/adirkuhn/commerceos/domain/locale"

// layoutTexts are the labels the default email blocks share.
type layoutTexts struct {
	greeting, order, product, quantity, total, subtotal, shipping, orderTotal, shippingTo, track string
}

// messageTexts are one default's own texts; text is the WhatsApp/SMS body.
type messageTexts struct {
	name, subject, heading, intro, text string
}

type defaultTexts struct {
	layout                                                                   layoutTexts
	orderConfirmation, orderStatus, orderShipped, refundIssued, refundReview messageTexts
}

var defaultTextsByLocale = map[locale.Locale]defaultTexts{
	locale.English: {
		layout: layoutTexts{
			greeting: "Hi {{customer_first_name}},", order: "Order", product: "Product", quantity: "Qty", total: "Total",
			subtotal: "Subtotal", shipping: "Shipping", orderTotal: "Total", shippingTo: "Shipping to", track: "Track your package",
		},
		orderConfirmation: messageTexts{
			name: "Order confirmation", subject: "Order #{{order_number}} confirmed",
			heading: "Thank you for your order!",
			intro:   "We received your order #{{order_number}} and will start preparing it right away.",
			text: joinLines("Hi {{customer_first_name}}, thank you for your order #{{order_number}} at {{store_name}}!",
				"Total: {{order_total}}. We'll let you know when it ships."),
		},
		orderStatus: messageTexts{
			name: "Order status update", subject: "Order #{{order_number}}: {{order_status_label}}",
			heading: "Your order is now: {{order_status_label}}",
			intro:   "The status of your order #{{order_number}} changed to {{order_status_label}}.",
			text:    "Hi {{customer_first_name}}, your order #{{order_number}} at {{store_name}} is now: {{order_status_label}}.",
		},
		orderShipped: messageTexts{
			name: "Order shipped", subject: "Order #{{order_number}} is on its way",
			heading: "Your order is on its way!",
			intro:   "Good news: your order #{{order_number}} has been handed to {{carrier}}.",
			text: joinLines("Hi {{customer_first_name}}, your order #{{order_number}} from {{store_name}} is on its way with {{carrier}}.",
				"Track it here: {{tracking_url}}"),
		},
		refundIssued: messageTexts{
			name: "Refund issued", subject: "Refund for order #{{order_number}}",
			heading: "We refunded {{refund_amount}}",
			intro:   "We refunded {{refund_amount}} for your order #{{order_number}} using your original payment method ({{payment_method}}). It can take a few days to show up on your account.",
			text:    "Hi {{customer_first_name}}, we refunded {{refund_amount}} for your order #{{order_number}} at {{store_name}}. It can take a few days to show up.",
		},
		refundReview: messageTexts{
			name: "Refund under review", subject: "We're reviewing the refund for order #{{order_number}}",
			heading: "Your refund is being reviewed",
			intro:   "We're checking the refund of {{refund_amount}} for your order #{{order_number}} and will get back to you soon.",
			text:    "Hi {{customer_first_name}}, we're checking the refund of {{refund_amount}} for your order #{{order_number}} at {{store_name}} and will get back to you soon.",
		},
	},
	locale.Portuguese: {
		layout: layoutTexts{
			greeting: "Olá {{customer_first_name}},", order: "Pedido", product: "Produto", quantity: "Qtd", total: "Total",
			subtotal: "Subtotal", shipping: "Frete", orderTotal: "Total", shippingTo: "Entregar em", track: "Rastrear seu pacote",
		},
		orderConfirmation: messageTexts{
			name: "Confirmação do pedido", subject: "Pedido #{{order_number}} confirmado",
			heading: "Obrigado pelo seu pedido!",
			intro:   "Recebemos seu pedido #{{order_number}} e já vamos começar a prepará-lo.",
			text: joinLines("Olá {{customer_first_name}}, obrigado pelo seu pedido #{{order_number}} na {{store_name}}!",
				"Total: {{order_total}}. Avisaremos quando ele for enviado."),
		},
		orderStatus: messageTexts{
			name: "Atualização do pedido", subject: "Pedido #{{order_number}}: {{order_status_label}}",
			heading: "Seu pedido agora está: {{order_status_label}}",
			intro:   "O status do seu pedido #{{order_number}} mudou para {{order_status_label}}.",
			text:    "Olá {{customer_first_name}}, seu pedido #{{order_number}} na {{store_name}} agora está: {{order_status_label}}.",
		},
		orderShipped: messageTexts{
			name: "Pedido enviado", subject: "Pedido #{{order_number}} está a caminho",
			heading: "Seu pedido está a caminho!",
			intro:   "Boa notícia: seu pedido #{{order_number}} foi entregue à {{carrier}}.",
			text: joinLines("Olá {{customer_first_name}}, seu pedido #{{order_number}} da {{store_name}} está a caminho com a {{carrier}}.",
				"Rastreie aqui: {{tracking_url}}"),
		},
		refundIssued: messageTexts{
			name: "Reembolso efetuado", subject: "Reembolso do pedido #{{order_number}}",
			heading: "Reembolsamos {{refund_amount}}",
			intro:   "Reembolsamos {{refund_amount}} do seu pedido #{{order_number}} no meio de pagamento original ({{payment_method}}). Pode levar alguns dias para aparecer na sua conta.",
			text:    "Olá {{customer_first_name}}, reembolsamos {{refund_amount}} do seu pedido #{{order_number}} na {{store_name}}. Pode levar alguns dias para aparecer.",
		},
		refundReview: messageTexts{
			name: "Reembolso em análise", subject: "Estamos analisando o reembolso do pedido #{{order_number}}",
			heading: "Seu reembolso está em análise",
			intro:   "Estamos verificando o reembolso de {{refund_amount}} do seu pedido #{{order_number}} e retornaremos em breve.",
			text:    "Olá {{customer_first_name}}, estamos verificando o reembolso de {{refund_amount}} do seu pedido #{{order_number}} na {{store_name}} e retornaremos em breve.",
		},
	},
	locale.Dutch: {
		layout: layoutTexts{
			greeting: "Hoi {{customer_first_name}},", order: "Bestelling", product: "Product", quantity: "Aantal", total: "Totaal",
			subtotal: "Subtotaal", shipping: "Verzending", orderTotal: "Totaal", shippingTo: "Verzenden naar", track: "Volg je pakket",
		},
		orderConfirmation: messageTexts{
			name: "Orderbevestiging", subject: "Bestelling #{{order_number}} bevestigd",
			heading: "Bedankt voor je bestelling!",
			intro:   "We hebben je bestelling #{{order_number}} ontvangen en gaan hem meteen klaarmaken.",
			text: joinLines("Hoi {{customer_first_name}}, bedankt voor je bestelling #{{order_number}} bij {{store_name}}!",
				"Totaal: {{order_total}}. We laten je weten wanneer hij verzonden is."),
		},
		orderStatus: messageTexts{
			name: "Statusupdate bestelling", subject: "Bestelling #{{order_number}}: {{order_status_label}}",
			heading: "Je bestelling is nu: {{order_status_label}}",
			intro:   "De status van je bestelling #{{order_number}} is gewijzigd naar {{order_status_label}}.",
			text:    "Hoi {{customer_first_name}}, je bestelling #{{order_number}} bij {{store_name}} is nu: {{order_status_label}}.",
		},
		orderShipped: messageTexts{
			name: "Bestelling verzonden", subject: "Bestelling #{{order_number}} is onderweg",
			heading: "Je bestelling is onderweg!",
			intro:   "Goed nieuws: je bestelling #{{order_number}} is overgedragen aan {{carrier}}.",
			text: joinLines("Hoi {{customer_first_name}}, je bestelling #{{order_number}} van {{store_name}} is onderweg met {{carrier}}.",
				"Volg hem hier: {{tracking_url}}"),
		},
		refundIssued: messageTexts{
			name: "Terugbetaling uitgevoerd", subject: "Terugbetaling voor bestelling #{{order_number}}",
			heading: "We hebben {{refund_amount}} terugbetaald",
			intro:   "We hebben {{refund_amount}} voor je bestelling #{{order_number}} terugbetaald via je oorspronkelijke betaalmethode ({{payment_method}}). Het kan een paar dagen duren voordat je het op je rekening ziet.",
			text:    "Hoi {{customer_first_name}}, we hebben {{refund_amount}} voor je bestelling #{{order_number}} bij {{store_name}} terugbetaald. Het kan een paar dagen duren voordat je het ziet.",
		},
		refundReview: messageTexts{
			name: "Terugbetaling in behandeling", subject: "We bekijken de terugbetaling voor bestelling #{{order_number}}",
			heading: "Je terugbetaling wordt bekeken",
			intro:   "We controleren de terugbetaling van {{refund_amount}} voor je bestelling #{{order_number}} en laten snel van ons horen.",
			text:    "Hoi {{customer_first_name}}, we controleren de terugbetaling van {{refund_amount}} voor je bestelling #{{order_number}} bij {{store_name}} en laten snel van ons horen.",
		},
	},
}
