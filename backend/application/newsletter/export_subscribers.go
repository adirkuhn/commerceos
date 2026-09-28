package newsletter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const exportBatch = 500

// ExportSubscribersUseCase writes confirmed subscribers as CSV (for an
// email tool). Only confirmed addresses: they're the ones that consented.
type ExportSubscribersUseCase struct{ deps Deps }

func NewExportSubscribersUseCase(deps Deps) *ExportSubscribersUseCase {
	return &ExportSubscribersUseCase{deps: deps}
}

// cell neutralises spreadsheet formula injection (a value starting with
// = + - @ would be executed by Excel/Sheets).
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func (uc *ExportSubscribersUseCase) Execute(ctx context.Context, out io.Writer) error {
	w := csv.NewWriter(out)
	if err := w.Write([]string{"email", "confirmed_at"}); err != nil {
		return fmt.Errorf("exportSubscribers: %w", err)
	}
	confirmed := newsletter.StatusConfirmed
	for offset := 0; ; offset += exportBatch {
		var batch []newsletter.Subscriber
		err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
			var err error
			batch, _, err = uc.deps.Repo.List(ctx, newsletter.ListFilter{Status: &confirmed, Limit: exportBatch, Offset: offset})
			return err
		})
		if err != nil {
			return fmt.Errorf("exportSubscribers: %w", err)
		}
		for _, s := range batch {
			at := ""
			if s.ConfirmedAt != nil {
				at = s.ConfirmedAt.UTC().Format(time.RFC3339)
			}
			if err := w.Write([]string{cell(s.Email), at}); err != nil {
				return fmt.Errorf("exportSubscribers: %w", err)
			}
		}
		if len(batch) < exportBatch {
			break
		}
	}
	w.Flush()
	return w.Error()
}
