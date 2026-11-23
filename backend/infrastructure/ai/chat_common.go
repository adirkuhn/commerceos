package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// Shared by every assistant.ChatModel adapter.

// assistantChatMaxTokens bounds one assistant step -- a final answer or a
// handful of tool calls, never a long document.
const assistantChatMaxTokens = 2048

// tooManyStepsText is how an app-written NoticeTooManySteps message reads
// to the model on the next turn (the user sees it translated instead).
const tooManyStepsText = "I needed too many lookups without reaching an answer, so I stopped and asked the user to add detail."

// assistantText is what an assistant message says to the model.
func assistantText(m assistant.Message) string {
	if m.Notice == assistant.NoticeTooManySteps {
		return tooManyStepsText
	}
	return m.Content
}

// busyStatus: rate limit, quota or overload on the provider's side --
// worth waiting in the queue for. 529 is Anthropic's "overloaded".
func busyStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		return true
	}
	return false
}

// chatError wraps a provider failure for the assistant, naming the
// provider and model so a retired model or a bad key is obvious in the
// request's last error. Transient ones (busyStatus, or a network failure)
// wrap ErrModelBusy so the request waits in the queue.
func chatError(label, model string, status int, err error) error {
	var netErr net.Error
	if busyStatus(status) || errors.As(err, &netErr) {
		return fmt.Errorf("%s (%s): %w: %w", label, model, assistant.ErrModelBusy, err)
	}
	return fmt.Errorf("%s (%s): %w", label, model, err)
}

// argsMap decodes a tool call's JSON arguments for providers that take
// them as an object; anything unparsable becomes {}.
func argsMap(arguments string) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal([]byte(arguments), &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

// generatedCallID names a tool call for providers that don't (Gemini,
// Ollama): unique within one reply, which is all a turn needs.
func generatedCallID(i int, name string) string {
	return fmt.Sprintf("call_%d_%s", i, name)
}
