package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func pendingBolConfirmation(id, tenantID, connectionID, processStatusID string) channelsDomain.BolShipmentConfirmation {
	return channelsDomain.BolShipmentConfirmation{
		ID: id, TenantID: tenantID, SalesOrderID: "so-" + id, ConnectionID: connectionID, ExternalOrderID: "C-" + id,
		ProcessStatusID: processStatusID, Status: channelsDomain.BolConfirmationPending,
	}
}

func bolStatuses(byID map[string]integrationsDomain.BolProcessStatus) func(context.Context, string) (integrationsDomain.BolProcessStatus, error) {
	return func(_ context.Context, id string) (integrationsDomain.BolProcessStatus, error) {
		s, ok := byID[id]
		if !ok {
			return integrationsDomain.BolProcessStatus{}, errors.New("bol: request failed (404)")
		}
		return s, nil
	}
}

func runCheck(t *testing.T, repo *fakeBolConfirmationRepo, source *fakeBolClientSource, outbox *eventrecordtest.Outbox) error {
	t.Helper()
	uc := appchannels.NewCheckBolShipmentConfirmationsUseCase(newMockGormDB(t, 10), repo, source, outbox, func() time.Time { return confirmNow })
	return uc.Run(context.Background())
}

func TestCheckBolShipmentConfirmations_resolvesFinishedOnes_leavesPendingOnes(t *testing.T) {
	repo := &fakeBolConfirmationRepo{pending: []channelsDomain.BolShipmentConfirmation{
		pendingBolConfirmation("ok", "tenant-1", "conn-1", "ps-ok"),
		pendingBolConfirmation("bad", "tenant-2", "conn-2", "ps-bad"),
		pendingBolConfirmation("wait", "tenant-1", "conn-1", "ps-wait"),
	}}
	bol := &fakeBolClient{getProcessStatusFn: bolStatuses(map[string]integrationsDomain.BolProcessStatus{
		"ps-ok":   {ID: "ps-ok", Status: "SUCCESS"},
		"ps-bad":  {ID: "ps-bad", Status: "FAILURE", ErrorMessage: "Invalid track and trace code."},
		"ps-wait": {ID: "ps-wait", Status: "PENDING"},
	})}
	source := &fakeBolClientSource{client: bol}
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, runCheck(t, repo, source, outbox))

	require.Len(t, repo.saved, 2)
	assert.Equal(t, "ok", repo.saved[0].ID)
	assert.Equal(t, channelsDomain.BolConfirmationSuccess, repo.saved[0].Status)
	require.NotNil(t, repo.saved[0].ResolvedAt)
	assert.Equal(t, confirmNow, *repo.saved[0].ResolvedAt)
	assert.Equal(t, "bad", repo.saved[1].ID)
	assert.Equal(t, channelsDomain.BolConfirmationFailure, repo.saved[1].Status)
	assert.Equal(t, "Invalid track and trace code.", repo.saved[1].ErrorMessage)

	assert.Equal(t, []string{"conn-1", "conn-2"}, source.connectionID, "one client per connection")

	require.Len(t, outbox.Events, 2)
	assert.Equal(t, "channels.bol_shipment_confirmation.resolved", outbox.Events[1].Subject)
	assert.Equal(t, "bad", outbox.Events[1].Payload["entity_id"])
	assert.Equal(t, "tenant-2", outbox.Events[1].Payload["tenant_id"])
	assert.Equal(t, [2]any{"pending", "failure"}, outbox.Events[1].Changes()["status"])
	assert.Equal(t, "Invalid track and trace code.", outbox.Events[1].Data()["error_message"])
	assert.Equal(t, "so-bad", outbox.Events[1].Data()["sales_order_id"])
}

func TestCheckBolShipmentConfirmations_oneFailingLookup_doesNotStopTheRest(t *testing.T) {
	repo := &fakeBolConfirmationRepo{pending: []channelsDomain.BolShipmentConfirmation{
		pendingBolConfirmation("missing", "tenant-1", "conn-1", "ps-missing"),
		pendingBolConfirmation("ok", "tenant-1", "conn-1", "ps-ok"),
	}}
	bol := &fakeBolClient{getProcessStatusFn: bolStatuses(map[string]integrationsDomain.BolProcessStatus{
		"ps-ok": {ID: "ps-ok", Status: "SUCCESS"},
	})}

	require.NoError(t, runCheck(t, repo, &fakeBolClientSource{client: bol}, &eventrecordtest.Outbox{}))

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "ok", repo.saved[0].ID)
}

func TestCheckBolShipmentConfirmations_connectionUnusable_skipsItsConfirmations(t *testing.T) {
	repo := &fakeBolConfirmationRepo{pending: []channelsDomain.BolShipmentConfirmation{
		pendingBolConfirmation("a", "tenant-1", "conn-1", "ps-a"),
	}}

	require.NoError(t, runCheck(t, repo, &fakeBolClientSource{err: errors.New("connection deleted")}, &eventrecordtest.Outbox{}))

	assert.Empty(t, repo.saved)
}

func TestCheckBolShipmentConfirmations_unknownStatusFromBol_isLeftPending(t *testing.T) {
	repo := &fakeBolConfirmationRepo{pending: []channelsDomain.BolShipmentConfirmation{
		pendingBolConfirmation("a", "tenant-1", "conn-1", "ps-a"),
	}}
	bol := &fakeBolClient{getProcessStatusFn: bolStatuses(map[string]integrationsDomain.BolProcessStatus{
		"ps-a": {ID: "ps-a", Status: "SOMETHING_NEW"},
	})}

	require.NoError(t, runCheck(t, repo, &fakeBolClientSource{client: bol}, &eventrecordtest.Outbox{}))

	assert.Empty(t, repo.saved)
}

func TestCheckBolShipmentConfirmations_eventCannotBeRecorded_runKeepsGoing(t *testing.T) {
	repo := &fakeBolConfirmationRepo{pending: []channelsDomain.BolShipmentConfirmation{
		pendingBolConfirmation("a", "tenant-1", "conn-1", "ps-a"),
	}}
	bol := &fakeBolClient{getProcessStatusFn: bolStatuses(map[string]integrationsDomain.BolProcessStatus{
		"ps-a": {ID: "ps-a", Status: "SUCCESS"},
	})}

	require.NoError(t, runCheck(t, repo, &fakeBolClientSource{client: bol}, &eventrecordtest.Outbox{Err: errors.New("outbox down")}),
		"logged per confirmation; the next run tries again")
}

func TestCheckBolShipmentConfirmations_cannotListPending_fails(t *testing.T) {
	repo := &fakeBolConfirmationRepo{listErr: errors.New("db down")}

	require.Error(t, runCheck(t, repo, &fakeBolClientSource{}, &eventrecordtest.Outbox{}))
}
