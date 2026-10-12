package settings_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubAppearanceRepo struct {
	saved     *settingsDomain.Appearance
	getErr    error
	upsertErr error
	upserted  []settingsDomain.Appearance
}

func (s *stubAppearanceRepo) Get(context.Context) (*settingsDomain.Appearance, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.saved, nil
}

func (s *stubAppearanceRepo) Upsert(_ context.Context, a *settingsDomain.Appearance) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *a)
	return nil
}

func appearanceInput() settingsDomain.AppearanceInput {
	return settingsDomain.AppearanceInput{
		Font: "inter", AccentColor: "#2f33b8", SidebarColor: "#14163F",
		Corners: "round", TextSize: "normal", Density: "compact",
	}
}

func TestGetAppearance_neverSavedReturnsDefaultsForTheTenant(t *testing.T) {
	uc := appsettings.NewGetAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{})

	a, err := uc.Execute(companyCtx())

	require.NoError(t, err)
	assert.Equal(t, settingsDomain.DefaultAppearance("tenant-1"), *a)
	assert.True(t, a.UpdatedAt.IsZero())
}

func TestGetAppearance_returnsSaved(t *testing.T) {
	saved := &settingsDomain.Appearance{TenantID: "tenant-1", Font: settingsDomain.FontBarlow, UpdatedAt: time.Now()}
	uc := appsettings.NewGetAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{saved: saved})

	a, err := uc.Execute(companyCtx())

	require.NoError(t, err)
	assert.Equal(t, settingsDomain.FontBarlow, a.Font)
}

func TestGetAppearance_propagatesRepoError(t *testing.T) {
	uc := appsettings.NewGetAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{getErr: errBoom})

	_, err := uc.Execute(companyCtx())

	assert.ErrorIs(t, err, errBoom)
}

func TestUpdateAppearance_firstSaveRecordsChangesAgainstTheDefaults(t *testing.T) {
	repo := &stubAppearanceRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), repo, outbox)

	a, err := uc.Execute(companyCtx(), appearanceInput())

	require.NoError(t, err)
	assert.Equal(t, settingsDomain.HexColor("#2F33B8"), a.AccentColor)
	assert.False(t, a.UpdatedAt.IsZero())
	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "tenant-1", repo.upserted[0].TenantID)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "settings.appearance.updated", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{
		"font":    {"atkinson", "inter"},
		"corners": {"normal", "round"},
		"density": {"comfortable", "compact"},
	}, e.Changes())
}

func TestUpdateAppearance_recordsOnlyFieldsThatChanged(t *testing.T) {
	before := settingsDomain.DefaultAppearance("tenant-1")
	before.Font = settingsDomain.FontInter
	before.Corners = settingsDomain.CornersRound
	before.Density = settingsDomain.DensityCompact
	before.UpdatedAt = time.Now().Add(-time.Hour)
	outbox := &eventrecordtest.Outbox{}
	in := appearanceInput()
	in.SidebarColor = "#000000"
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{saved: &before}, outbox)

	_, err := uc.Execute(companyCtx(), in)

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"sidebar_color": {"#14163F", "#000000"}}, e.Changes())
}

func TestUpdateAppearance_invalidInputWritesNothing(t *testing.T) {
	repo := &stubAppearanceRepo{}
	outbox := &eventrecordtest.Outbox{}
	in := appearanceInput()
	in.AccentColor = "#FFFF00"
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(companyCtx(), in)

	assert.ErrorIs(t, err, settingsDomain.ErrAppearanceAccentLowContrast)
	assert.Empty(t, repo.upserted)
	assert.Empty(t, outbox.Events)
}

func TestUpdateAppearance_propagatesGetError(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{getErr: errBoom}, outbox)

	_, err := uc.Execute(companyCtx(), appearanceInput())

	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, outbox.Events)
}

func TestUpdateAppearance_propagatesUpsertError(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{upsertErr: errBoom}, outbox)

	_, err := uc.Execute(companyCtx(), appearanceInput())

	assert.ErrorIs(t, err, errBoom)
	assert.Empty(t, outbox.Events)
}

func TestUpdateAppearance_failsWhenTheEventCantBeRecorded(t *testing.T) {
	uc := appsettings.NewUpdateAppearanceUseCase(newMockGormDB(t), &stubAppearanceRepo{}, &eventrecordtest.Outbox{Err: errBoom})

	_, err := uc.Execute(companyCtx(), appearanceInput())

	assert.ErrorIs(t, err, errBoom)
}
