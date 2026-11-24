package settings

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrInvalidAppearanceLook     = errs.New("APPEARANCE_INVALID_LOOK", "default look must be wallet or graphite")
	ErrInvalidAppearanceCorners  = errs.New("APPEARANCE_INVALID_CORNERS", "corners must be sharp, normal or round")
	ErrInvalidAppearanceTextSize = errs.New("APPEARANCE_INVALID_TEXT_SIZE", "text size must be normal, large or larger")
	ErrInvalidAppearanceDensity  = errs.New("APPEARANCE_INVALID_DENSITY", "density must be compact or comfortable")
)

// AppearanceLook is one of the admin's looks (each with light and dark).
type AppearanceLook string

const (
	LookWallet   AppearanceLook = "wallet" // Ticket Wallet, the default
	LookGraphite AppearanceLook = "graphite"
)

type AppearanceCorners string

const (
	CornersSharp  AppearanceCorners = "sharp"
	CornersNormal AppearanceCorners = "normal"
	CornersRound  AppearanceCorners = "round"
)

type AppearanceTextSize string

const (
	TextSizeNormal AppearanceTextSize = "normal"
	TextSizeLarge  AppearanceTextSize = "large"
	TextSizeLarger AppearanceTextSize = "larger"
)

type AppearanceDensity string

const (
	DensityCompact     AppearanceDensity = "compact"
	DensityComfortable AppearanceDensity = "comfortable"
)

// Appearance is how the admin looks for everyone in a shop. Separate from
// TenantSettings: cosmetic, a different reason to change. DefaultLook is
// what everyone sees unless they pick another look in their own browser.
// A zero UpdatedAt means it was never saved (the defaults).
type Appearance struct {
	TenantID    string
	DefaultLook AppearanceLook
	Corners     AppearanceCorners
	TextSize    AppearanceTextSize
	Density     AppearanceDensity
	UpdatedAt   time.Time
}

// AppearanceInput is what a caller may set; NewAppearance validates it.
type AppearanceInput struct {
	DefaultLook string
	Corners     string
	TextSize    string
	Density     string
}

type AppearanceRepository interface {
	// Get returns the tenant's saved appearance, or nil if it was never
	// saved (callers fall back to DefaultAppearance).
	Get(ctx context.Context) (*Appearance, error)
	Upsert(ctx context.Context, a *Appearance) error
}

// DefaultAppearance is what a shop gets until it saves its own.
func DefaultAppearance(tenantID string) Appearance {
	return Appearance{
		TenantID: tenantID, DefaultLook: LookWallet,
		Corners: CornersNormal, TextSize: TextSizeNormal, Density: DensityComfortable,
	}
}

func NewAppearance(tenantID string, in AppearanceInput, now time.Time) (*Appearance, error) {
	look, err := parseLook(in.DefaultLook)
	if err != nil {
		return nil, err
	}
	corners, err := parseCorners(in.Corners)
	if err != nil {
		return nil, err
	}
	textSize, err := parseTextSize(in.TextSize)
	if err != nil {
		return nil, err
	}
	density, err := parseDensity(in.Density)
	if err != nil {
		return nil, err
	}
	return &Appearance{
		TenantID: tenantID, DefaultLook: look,
		Corners: corners, TextSize: textSize, Density: density, UpdatedAt: now,
	}, nil
}

func parseLook(s string) (AppearanceLook, error) {
	switch l := AppearanceLook(s); l {
	case LookWallet, LookGraphite:
		return l, nil
	}
	return "", ErrInvalidAppearanceLook
}

func parseCorners(s string) (AppearanceCorners, error) {
	switch c := AppearanceCorners(s); c {
	case CornersSharp, CornersNormal, CornersRound:
		return c, nil
	}
	return "", ErrInvalidAppearanceCorners
}

func parseTextSize(s string) (AppearanceTextSize, error) {
	switch t := AppearanceTextSize(s); t {
	case TextSizeNormal, TextSizeLarge, TextSizeLarger:
		return t, nil
	}
	return "", ErrInvalidAppearanceTextSize
}

func parseDensity(s string) (AppearanceDensity, error) {
	switch d := AppearanceDensity(s); d {
	case DensityCompact, DensityComfortable:
		return d, nil
	}
	return "", ErrInvalidAppearanceDensity
}
