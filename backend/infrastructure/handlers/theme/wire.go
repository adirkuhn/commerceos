package theme

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"gorm.io/gorm"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/presetimages"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/themecatalog"
)

// NewUseCases builds every theme use case on the real adapters. The
// component catalog comes from the embedded theme contract.
func NewUseCases(db *gorm.DB, storage theme.AssetStorage) (UseCases, error) {
	spec, err := gentheme.GetSpec()
	if err != nil {
		return UseCases{}, fmt.Errorf("theme: load openapi spec: %w", err)
	}
	catalog, err := themecatalog.FromSpec(spec)
	if err != nil {
		return UseCases{}, err
	}
	channels := repositories.NewSalesChannelRepo(db)
	d := apptheme.Deps{
		DB: db, Channels: channels, Themes: repositories.NewThemeRepo(db), Versions: repositories.NewThemeVersionRepo(db),
		Docs: repositories.NewThemeDocumentRepo(db), Sessions: repositories.NewThemeSessionRepo(db),
		Outbox: repositories.NewOutboxRepository(db), Catalog: catalog, NewToken: newToken, Now: time.Now,
		Assets: repositories.NewThemeAssetRepo(db), Storage: storage, PresetImages: presetimages.Files{},
	}
	return UseCases{
		ListThemes: apptheme.NewListThemesUseCase(d), CreateTheme: apptheme.NewCreateThemeUseCase(d),
		GetTheme: apptheme.NewGetThemeUseCase(d), RenameTheme: apptheme.NewRenameThemeUseCase(d),
		DeleteTheme: apptheme.NewDeleteThemeUseCase(d), ListVersions: apptheme.NewListVersionsUseCase(d),
		PublishTheme: apptheme.NewPublishThemeUseCase(d), RollbackTheme: apptheme.NewRollbackThemeUseCase(d),
		CreateSession: apptheme.NewCreateSessionUseCase(d), ResolveSession: apptheme.NewResolveSessionUseCase(d),
		OpenDraft: apptheme.NewOpenDraftUseCase(d), SaveDraftSettings: apptheme.NewSaveDraftSettingsUseCase(d),
		GetDraftDocument: apptheme.NewGetDraftDocumentUseCase(d), SaveDraftDocument: apptheme.NewSaveDraftDocumentUseCase(d),
		CreatePage: apptheme.NewCreatePageUseCase(d), DeletePage: apptheme.NewDeletePageUseCase(d),
		GetLiveTheme: apptheme.NewGetLiveThemeUseCase(d), GetPublishedDocument: apptheme.NewGetPublishedDocumentUseCase(d),
		GetPreviewTheme: apptheme.NewGetPreviewThemeUseCase(d), GetPreviewDocument: apptheme.NewGetPreviewDocumentUseCase(d),
		ResolveStorefrontKey: appapikeys.NewResolveAPIKeyUseCase(db, repositories.NewAPIKeyRepo(db), channels),
		ListAssets:           apptheme.NewListAssetsUseCase(d), UploadAsset: apptheme.NewUploadAssetUseCase(d),
		UpdateAsset: apptheme.NewUpdateAssetUseCase(d), DeleteAsset: apptheme.NewDeleteAssetUseCase(d),
	}, nil
}

// newToken returns 32 random bytes, URL-safe base64 -- 256 bits, so
// guessing a live session token is not feasible.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
