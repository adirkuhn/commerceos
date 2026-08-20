package pms_test

import (
	"context"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
)

type stubDiscardListerForMarginInsight struct {
	result *invapp.ListDiscardsResult
	err    error
}

func (s *stubDiscardListerForMarginInsight) Execute(_ context.Context, _ invapp.ListDiscardsInput) (*invapp.ListDiscardsResult, error) {
	if s.result == nil && s.err == nil {
		return &invapp.ListDiscardsResult{}, nil
	}
	return s.result, s.err
}
