package deadletters_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/deadletters"
	"github.com/adirkuhn/commerceos/domain/deadletter"
)

func TestListDeadLetters_passesTheFilterToTheStore(t *testing.T) {
	store := storeWith()
	store.page = deadletter.Page{Items: []deadletter.DeadLetter{sampleDeadLetter()}, NextCursor: 8, Total: 3}
	uc := deadletters.NewListDeadLettersUseCase(store)
	f := deadletter.Filter{Stream: deadletter.StreamEvents, Subject: "oms.sales_order.status_changed", TenantID: "t-1", After: 2, Limit: 10}

	page, err := uc.Execute(superAdmin(), f)

	require.NoError(t, err)
	assert.Equal(t, f, store.gotFilter)
	assert.Equal(t, store.page, page)
}

func TestListDeadLetters_rejectsUnknownStream(t *testing.T) {
	uc := deadletters.NewListDeadLettersUseCase(storeWith())

	_, err := uc.Execute(superAdmin(), deadletter.Filter{Stream: "orders"})

	assert.ErrorIs(t, err, deadletter.ErrInvalidStream)
}

func TestListDeadLetters_onlySuperAdmin(t *testing.T) {
	uc := deadletters.NewListDeadLettersUseCase(storeWith())

	_, err := uc.Execute(context.Background(), deadletter.Filter{Stream: deadletter.StreamEvents})

	assert.ErrorIs(t, err, deadletter.ErrForbidden)
}

func TestListDeadLetters_storeErrorIsWrapped(t *testing.T) {
	store := storeWith()
	store.listErr = errBoom
	uc := deadletters.NewListDeadLettersUseCase(store)

	_, err := uc.Execute(superAdmin(), deadletter.Filter{Stream: deadletter.StreamEvents})

	assert.ErrorIs(t, err, errBoom)
}

func TestGetDeadLetter(t *testing.T) {
	dl := sampleDeadLetter()
	uc := deadletters.NewGetDeadLetterUseCase(storeWith(dl))

	got, err := uc.Execute(superAdmin(), deadletter.StreamEvents, 7)

	require.NoError(t, err)
	assert.Equal(t, dl, got)
}

func TestGetDeadLetter_notFound(t *testing.T) {
	uc := deadletters.NewGetDeadLetterUseCase(storeWith())

	_, err := uc.Execute(superAdmin(), deadletter.StreamEvents, 99)

	assert.ErrorIs(t, err, deadletter.ErrNotFound)
}

func TestGetDeadLetter_rejectsUnknownStream(t *testing.T) {
	uc := deadletters.NewGetDeadLetterUseCase(storeWith())

	_, err := uc.Execute(superAdmin(), "orders", 1)

	assert.ErrorIs(t, err, deadletter.ErrInvalidStream)
}

func TestGetDeadLetter_onlySuperAdmin(t *testing.T) {
	uc := deadletters.NewGetDeadLetterUseCase(storeWith(sampleDeadLetter()))

	_, err := uc.Execute(context.Background(), deadletter.StreamEvents, 7)

	assert.ErrorIs(t, err, deadletter.ErrForbidden)
}
