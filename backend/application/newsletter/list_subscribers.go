package newsletter

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListInput struct {
	Status  *newsletter.Status
	Search  string
	Page    int
	PerPage int
}

type ListSubscribersUseCase struct{ deps Deps }

func NewListSubscribersUseCase(deps Deps) *ListSubscribersUseCase {
	return &ListSubscribersUseCase{deps: deps}
}

func (uc *ListSubscribersUseCase) Execute(ctx context.Context, in ListInput) ([]newsletter.Subscriber, int64, error) {
	perPage, page := in.PerPage, in.Page
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	if page < 1 {
		page = 1
	}
	var list []newsletter.Subscriber
	var total int64
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		var err error
		list, total, err = uc.deps.Repo.List(ctx, newsletter.ListFilter{Status: in.Status, Search: in.Search, Limit: perPage, Offset: (page - 1) * perPage})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listSubscribers: %w", err)
	}
	return list, total, nil
}
