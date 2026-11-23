package events

// Admin AI assistant (backend/application/assistant). What the user wrote
// and what the model answered are never in an event -- read the request's
// messages by entity_id.
var (
	AssistantRequestCreated = defineAction("assistant.request.created",
		"An admin asked the AI assistant something; the request is queued for the model.")
	AssistantRequestReplied = defineAction("assistant.request.replied",
		"An admin answered the assistant's question or asked a follow-up; the request is queued again.")
	AssistantRequestAnswered = defineAction("assistant.request.answered",
		"The assistant's model answered a request. data: attempts, tool_calls.")
	AssistantRequestQuestionAsked = defineAction("assistant.request.question_asked",
		"The assistant's model asked the admin to clarify a request. data: attempts, tool_calls, notice.")
	AssistantRequestCancelled = defineAction("assistant.request.cancelled",
		"An admin cancelled an assistant request. data: previous_status.")
)

// Assistant proposals: changes the assistant proposed, waiting for a
// person's decision. What a confirm creates raises its own event (e.g.
// quotes.quote.created).
var (
	AssistantProposalCreated = defineAction("assistant.proposal.created",
		"The assistant proposed a change for an admin to confirm. data: request_id, kind.")
	AssistantProposalConfirmed = defineAction("assistant.proposal.confirmed",
		"An admin confirmed an assistant proposal and it ran. data: request_id, kind, result_id, result_ref.")
	AssistantProposalRejected = defineAction("assistant.proposal.rejected",
		"An admin rejected an assistant proposal. data: request_id, kind.")
)

// AssistantRequestRetried is raised when an admin asks a waiting request to
// be tried again now instead of at its next scheduled attempt.
var AssistantRequestRetried = defineAction("assistant.request.retried",
	"An admin asked a request waiting after a failed attempt to be tried again now. data: wait_reason.")
