package deadletters

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/deadletter"
)

type lister interface {
	List(ctx context.Context, f deadletter.Filter) (deadletter.Page, error)
}

// ListDeadLettersUseCase lists one stream's dead letters, filtered and
// paginated by sequence.
type ListDeadLettersUseCase struct{ store lister }

func NewListDeadLettersUseCase(store lister) *ListDeadLettersUseCase {
	return &ListDeadLettersUseCase{store: store}
}

func (uc *ListDeadLettersUseCase) Execute(ctx context.Context, f deadletter.Filter) (deadletter.Page, error) {
	if err := checkAccess(ctx, f.Stream); err != nil {
		return deadletter.Page{}, fmt.Errorf("listDeadLetters: %w", err)
	}
	page, err := uc.store.List(ctx, f)
	if err != nil {
		return deadletter.Page{}, fmt.Errorf("listDeadLetters: %w", err)
	}
	return page, nil
}
