package moneybird

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

// DocumentStatus is what a quote or order page shows about Moneybird:
// Document is nil until it was first sent. AdministrationID builds the
// link to the document in Moneybird.
type DocumentStatus struct {
	Document         *mb.Document
	AdministrationID string
	AutoSync         bool
}

// GetDocumentUseCase reads where a quote or order stands in Moneybird.
type GetDocumentUseCase struct {
	db       *gorm.DB
	conns    connectionLister
	settings mb.SettingsRepository
	recorder documentRecorder
	secret   string
}

func NewGetDocumentUseCase(db *gorm.DB, conns connectionLister, settings mb.SettingsRepository, docs mb.DocumentRepository, secret string) *GetDocumentUseCase {
	return &GetDocumentUseCase{db: db, conns: conns, settings: settings, recorder: documentRecorder{db: db, docs: docs}, secret: secret}
}

func (uc *GetDocumentUseCase) Execute(ctx context.Context, sourceType mb.SourceType, sourceID string) (*DocumentStatus, error) {
	acc, err := loadAccount(ctx, uc.db, uc.conns, uc.settings, uc.secret)
	if err != nil {
		return nil, fmt.Errorf("getMoneybirdDocument: %w", err)
	}
	doc, err := uc.recorder.find(ctx, sourceType, sourceID)
	if err != nil {
		return nil, fmt.Errorf("getMoneybirdDocument: %w", err)
	}
	return &DocumentStatus{Document: doc, AdministrationID: acc.settings.AdministrationID, AutoSync: acc.settings.AutoSync}, nil
}
