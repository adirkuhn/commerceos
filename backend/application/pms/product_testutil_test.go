package pms_test

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductHistoryRepo struct {
	created    []pmsDomain.ProductHistory
	createErr  error
	listResult []pmsDomain.ProductHistory
	listErr    error
}

func (s *stubProductHistoryRepo) Create(_ context.Context, h *pmsDomain.ProductHistory) error {
	if s.createErr != nil {
		return s.createErr
	}
	h.ID = "product-history-1"
	s.created = append(s.created, *h)
	return nil
}

func (s *stubProductHistoryRepo) ListByEntity(_ context.Context, _ pmsDomain.ProductHistoryEntityType, _ string) ([]pmsDomain.ProductHistory, error) {
	return s.listResult, s.listErr
}
