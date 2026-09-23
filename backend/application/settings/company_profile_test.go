package settings_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubCompanyRepo struct {
	got       *settingsDomain.CompanyProfile
	getErr    error
	upsertErr error
	upserted  []settingsDomain.CompanyProfile
}

func (s *stubCompanyRepo) Get(context.Context) (*settingsDomain.CompanyProfile, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.got != nil {
		return s.got, nil
	}
	return &settingsDomain.CompanyProfile{}, nil
}

func (s *stubCompanyRepo) Upsert(_ context.Context, p *settingsDomain.CompanyProfile) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *p)
	return nil
}

type enqueued struct {
	subject, eventID string
	payload          []byte
}

type stubEnqueuer struct {
	err  error
	rows []enqueued
}

func (s *stubEnqueuer) Enqueue(_ context.Context, subject, eventID string, payload []byte, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	s.rows = append(s.rows, enqueued{subject, eventID, payload})
	return nil
}

func companyCtx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

func TestGetCompanyProfile_returnsSavedProfile(t *testing.T) {
	repo := &stubCompanyRepo{got: &settingsDomain.CompanyProfile{TenantID: "tenant-1", LegalName: "Cantinho B.V."}}
	uc := appsettings.NewGetCompanyProfileUseCase(newMockGormDB(t), repo)

	p, err := uc.Execute(companyCtx())

	require.NoError(t, err)
	assert.Equal(t, "Cantinho B.V.", p.LegalName)
}

func TestGetCompanyProfile_emptyProfileCarriesTenant(t *testing.T) {
	uc := appsettings.NewGetCompanyProfileUseCase(newMockGormDB(t), &stubCompanyRepo{})

	p, err := uc.Execute(companyCtx())

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", p.TenantID)
	assert.Equal(t, "", p.LegalName)
}

func TestGetCompanyProfile_propagatesRepoError(t *testing.T) {
	uc := appsettings.NewGetCompanyProfileUseCase(newMockGormDB(t), &stubCompanyRepo{getErr: errBoom})

	_, err := uc.Execute(companyCtx())

	assert.True(t, errors.Is(err, errBoom))
}

func TestUpdateCompanyProfile_savesAndEnqueuesEvent(t *testing.T) {
	repo := &stubCompanyRepo{}
	outbox := &stubEnqueuer{}
	uc := appsettings.NewUpdateCompanyProfileUseCase(newMockGormDB(t), repo, outbox)

	p, err := uc.Execute(companyCtx(), settingsDomain.CompanyProfileInput{LegalName: " Cantinho B.V. ", VATNumber: "nl123456789b01"})

	require.NoError(t, err)
	assert.Equal(t, "Cantinho B.V.", p.LegalName)
	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "tenant-1", repo.upserted[0].TenantID)
	assert.Equal(t, "NL123456789B01", repo.upserted[0].VATNumber)

	require.Len(t, outbox.rows, 1)
	assert.Equal(t, "settings.company_profile.updated", outbox.rows[0].subject)
	var payload struct {
		EventID  string `json:"event_id"`
		TenantID string `json:"tenant_id"`
	}
	require.NoError(t, json.Unmarshal(outbox.rows[0].payload, &payload))
	assert.Equal(t, "tenant-1", payload.TenantID)
	assert.Equal(t, outbox.rows[0].eventID, payload.EventID)
	assert.NotEmpty(t, payload.EventID)
}

func TestUpdateCompanyProfile_invalidInputWritesNothing(t *testing.T) {
	repo := &stubCompanyRepo{}
	outbox := &stubEnqueuer{}
	uc := appsettings.NewUpdateCompanyProfileUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(companyCtx(), settingsDomain.CompanyProfileInput{LegalName: "X", IBAN: "NL00BAD"})

	assert.True(t, errors.Is(err, settingsDomain.ErrInvalidIBAN))
	assert.Empty(t, repo.upserted)
	assert.Empty(t, outbox.rows)
}

func TestUpdateCompanyProfile_propagatesRepoError(t *testing.T) {
	outbox := &stubEnqueuer{}
	uc := appsettings.NewUpdateCompanyProfileUseCase(newMockGormDB(t), &stubCompanyRepo{upsertErr: errBoom}, outbox)

	_, err := uc.Execute(companyCtx(), settingsDomain.CompanyProfileInput{LegalName: "Cantinho B.V."})

	assert.True(t, errors.Is(err, errBoom))
	assert.Empty(t, outbox.rows)
}

func TestUpdateCompanyProfile_propagatesOutboxError(t *testing.T) {
	uc := appsettings.NewUpdateCompanyProfileUseCase(newMockGormDB(t), &stubCompanyRepo{}, &stubEnqueuer{err: errBoom})

	_, err := uc.Execute(companyCtx(), settingsDomain.CompanyProfileInput{LegalName: "Cantinho B.V."})

	assert.True(t, errors.Is(err, errBoom))
}
