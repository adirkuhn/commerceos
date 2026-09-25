package whatsapp

import (
	"strings"
)

// evolutionAPIMessage is Evolution API's "Message" event webhook payload --
// CONFIRMED against a real running instance (2026-09-15, event ID
// 0000000000000000000000000000000A), replacing an earlier version built
// only from published docs that turned out wrong on every field path (it
// assumed a WhatsApp Business Cloud API-style "messages.upsert" shape --
// data.key.remoteJid/data.key.fromMe/data.message -- but this gateway is
// whatsmeow-based and sends its own Go event struct verbatim, unwrapped,
// with no json tags of its own: Data.Info.*/Data.Message.* using Go's
// exported-field casing, not lowerCamelCase). The old struct's explicit
// (wrong-case) json tags meant encoding/json's case-insensitive fallback
// never kicked in either, so every field silently zero-valued and every
// inbound message was dropped by the phone=="" guard in Execute -- this was
// not a LID-specific bug, no inbound message was ever parsed correctly.
//
// Sender is already whatsmeow's own LID-vs-phone-JID reconciliation
// result ("<phone>@s.whatsapp.net") -- SenderAlt carries the alternate
// "<numeric>@lid" identifier when the gateway detects a LID/phone-JID
// swap case, and must never be used for phone(), it is not a phone
// number. IsFromMe is true for a message this same WhatsApp number sent
// (including our own outbound replies echoed back by some gateway
// configurations) and must never be treated as an inbound command, or a
// reply could recursively trigger another reply. IsGroup is an explicit
// boolean from the gateway itself, used instead of string-matching the
// JID's "@g.us" suffix.
type evolutionAPIMessage struct {
	Data struct {
		Info struct {
			Sender   string `json:"Sender"`
			IsFromMe bool   `json:"IsFromMe"`
			IsGroup  bool   `json:"IsGroup"`
		} `json:"Info"`
		Message struct {
			Conversation        string `json:"conversation"`
			ExtendedTextMessage struct {
				Text string `json:"text"`
			} `json:"extendedTextMessage"`
		} `json:"Message"`
	} `json:"data"`
}

func (m evolutionAPIMessage) phone() string {
	return strings.SplitN(m.Data.Info.Sender, "@", 2)[0]
}

// isGroup is true for a group chat, as opposed to a direct 1-1 chat.
// Group messages are deliberately unsupported (no bot commands, no order
// flow, no customer lookup by "phone" -- a group has no single sender
// phone number) and must be dropped before Execute does anything with
// phone().
func (m evolutionAPIMessage) isGroup() bool {
	return m.Data.Info.IsGroup
}

func (m evolutionAPIMessage) text() string {
	if m.Data.Message.Conversation != "" {
		return m.Data.Message.Conversation
	}
	return m.Data.Message.ExtendedTextMessage.Text
}
