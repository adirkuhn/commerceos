package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	closeToExpiryDefaultPerPage = 50
	closeToExpiryMaxPerPage     = 200
)

// tenantSettingsGetter narrows settings.Repository to the read this
// package's close-to-expiry use cases need (ISP).
type tenantSettingsGetter interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

// ListCloseToExpiryInput picks one close-to-expiry window's list.
// WindowDays 0 means the largest window. Acknowledged lists the batches
// already marked as seen for it instead of the ones still to look at.
type ListCloseToExpiryInput struct {
	WindowDays   int
	Acknowledged bool
	Page         int
	PerPage      int
}

// CloseToExpiryWindow is one window's band, its calendar dates in the
// tenant's time zone and how many batches it holds.
type CloseToExpiryWindow struct {
	Band     settingsDomain.ExpiryBand
	From, To time.Time
	Counts   invDomain.CloseToExpiryCounts
}

type ListCloseToExpiryResult struct {
	// Windows are every window the tenant uses, largest first.
	Windows []CloseToExpiryWindow
	// Window is the one listed.
	Window CloseToExpiryWindow
	Items  []invDomain.CloseToExpiryBatch
	// Today is the tenant's calendar date the windows were measured from.
	Today   time.Time
	Page    int
	PerPage int
	// Total is the size of the listed (open or acknowledged) list.
	Total int
}

// ListCloseToExpiryBatchesUseCase is the close-to-expiry screen: batches
// with stock left, by window -- the same bands as the email report (see
// settingsDomain.ExpiryAlert.ReviewBands).
type ListCloseToExpiryBatchesUseCase struct {
	db       *gorm.DB
	settings tenantSettingsGetter
	reader   invDomain.CloseToExpiryReader
	now      func() time.Time
}

func NewListCloseToExpiryBatchesUseCase(db *gorm.DB, settings tenantSettingsGetter, reader invDomain.CloseToExpiryReader, now func() time.Time) *ListCloseToExpiryBatchesUseCase {
	return &ListCloseToExpiryBatchesUseCase{db: db, settings: settings, reader: reader, now: now}
}

func (uc *ListCloseToExpiryBatchesUseCase) Execute(ctx context.Context, in ListCloseToExpiryInput) (*ListCloseToExpiryResult, error) {
	page, perPage := closeToExpiryPaging(in.Page, in.PerPage)
	var res *ListCloseToExpiryResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.settings.Get(ctx)
		if err != nil {
			return err
		}
		today := tenantToday(uc.now(), s.Timezone)
		bands := s.ExpiryAlert.ReviewBands()
		selected, err := selectReviewBand(s.ExpiryAlert, bands, in.WindowDays)
		if err != nil {
			return err
		}

		res = &ListCloseToExpiryResult{Today: today, Page: page, PerPage: perPage, Windows: make([]CloseToExpiryWindow, len(bands))}
		for i, band := range bands {
			from, to := bandDates(band, today)
			counts, err := uc.reader.CountCloseToExpiryBatches(ctx, from, to, band.WindowDays)
			if err != nil {
				return err
			}
			res.Windows[i] = CloseToExpiryWindow{Band: band, From: from, To: to, Counts: counts}
			if band == selected {
				res.Window = res.Windows[i]
			}
		}
		res.Total = res.Window.Counts.Open
		if in.Acknowledged {
			res.Total = res.Window.Counts.Acknowledged
		}
		res.Items, err = uc.reader.ListCloseToExpiryBatches(ctx, invDomain.CloseToExpiryQuery{
			From: res.Window.From, To: res.Window.To, WindowDays: selected.WindowDays,
			Acknowledged: in.Acknowledged, Limit: perPage, Offset: (page - 1) * perPage,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCloseToExpiryBatches: %w", err)
	}
	return res, nil
}

// selectReviewBand is windowDays' band, or the largest one for 0.
func selectReviewBand(alert settingsDomain.ExpiryAlert, bands []settingsDomain.ExpiryBand, windowDays int) (settingsDomain.ExpiryBand, error) {
	if windowDays == 0 {
		return bands[0], nil
	}
	band, ok := alert.ReviewBand(windowDays)
	if !ok {
		return settingsDomain.ExpiryBand{}, invDomain.ErrUnknownExpiryWindow
	}
	return band, nil
}

// bandDates are the calendar dates a band covers, counted from today.
func bandDates(band settingsDomain.ExpiryBand, today time.Time) (from, to time.Time) {
	return today.AddDate(0, 0, band.MinDaysLeft), today.AddDate(0, 0, band.WindowDays)
}

func closeToExpiryPaging(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = closeToExpiryDefaultPerPage
	}
	return page, min(perPage, closeToExpiryMaxPerPage)
}
