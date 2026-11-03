package deadletters

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/deadletter"
)

// GetDeadLetterUseCase reads one dead letter, data and failure details.
type GetDeadLetterUseCase struct{ store getter }

func NewGetDeadLetterUseCase(store getter) *GetDeadLetterUseCase {
	return &GetDeadLetterUseCase{store: store}
}

func (uc *GetDeadLetterUseCase) Execute(ctx context.Context, stream deadletter.Stream, seq uint64) (deadletter.DeadLetter, error) {
	if err := checkAccess(ctx, stream); err != nil {
		return deadletter.DeadLetter{}, fmt.Errorf("getDeadLetter: %w", err)
	}
	d, err := uc.store.Get(ctx, stream, seq)
	if err != nil {
		return deadletter.DeadLetter{}, fmt.Errorf("getDeadLetter: %w", err)
	}
	return d, nil
}
