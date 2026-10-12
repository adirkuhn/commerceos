package settings

import (
	"context"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrInvalidAppearanceFont       = errs.New("APPEARANCE_INVALID_FONT", "font must be atkinson, inter, ibm_plex_sans, source_sans or barlow")
	ErrInvalidAppearanceColor      = errs.New("APPEARANCE_INVALID_COLOR", "colours must be written as #RRGGBB")
	ErrAppearanceAccentLowContrast = errs.New("APPEARANCE_ACCENT_LOW_CONTRAST", "the accent colour must reach a 4.5:1 contrast ratio with white text")
	ErrInvalidAppearanceCorners    = errs.New("APPEARANCE_INVALID_CORNERS", "corners must be sharp, normal or round")
	ErrInvalidAppearanceTextSize   = errs.New("APPEARANCE_INVALID_TEXT_SIZE", "text size must be normal, large or larger")
	ErrInvalidAppearanceDensity    = errs.New("APPEARANCE_INVALID_DENSITY", "density must be compact or comfortable")
)

type AppearanceFont string

const (
	FontAtkinson    AppearanceFont = "atkinson"
	FontInter       AppearanceFont = "inter"
	FontIBMPlexSans AppearanceFont = "ibm_plex_sans"
	FontSourceSans  AppearanceFont = "source_sans"
	FontBarlow      AppearanceFont = "barlow"
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

// HexColor is an upper-case "#RRGGBB" colour (see ParseHexColor).
type HexColor string

const (
	defaultAccentColor  HexColor = "#2F33B8"
	defaultSidebarColor HexColor = "#14163F"
	white               HexColor = "#FFFFFF"
)

// MinAccentContrast is WCAG 2.x AA for normal text: filled buttons put
// white text on the accent colour.
const MinAccentContrast = 4.5

// Appearance is how the admin looks for everyone in a shop (the new design
// only). Separate from TenantSettings: cosmetic, a different reason to
// change. A zero UpdatedAt means it was never saved (the defaults).
type Appearance struct {
	TenantID     string
	Font         AppearanceFont
	AccentColor  HexColor
	SidebarColor HexColor
	Corners      AppearanceCorners
	TextSize     AppearanceTextSize
	Density      AppearanceDensity
	UpdatedAt    time.Time
}

// AppearanceInput is what a caller may set; NewAppearance validates and
// normalises it.
type AppearanceInput struct {
	Font         string
	AccentColor  string
	SidebarColor string
	Corners      string
	TextSize     string
	Density      string
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
		TenantID: tenantID, Font: FontAtkinson, AccentColor: defaultAccentColor, SidebarColor: defaultSidebarColor,
		Corners: CornersNormal, TextSize: TextSizeNormal, Density: DensityComfortable,
	}
}

func NewAppearance(tenantID string, in AppearanceInput, now time.Time) (*Appearance, error) {
	font, err := parseFont(in.Font)
	if err != nil {
		return nil, err
	}
	accent, err := ParseHexColor(in.AccentColor)
	if err != nil {
		return nil, err
	}
	sidebar, err := ParseHexColor(in.SidebarColor)
	if err != nil {
		return nil, err
	}
	if ContrastRatio(accent, white) < MinAccentContrast {
		return nil, ErrAppearanceAccentLowContrast
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
		TenantID: tenantID, Font: font, AccentColor: accent, SidebarColor: sidebar,
		Corners: corners, TextSize: textSize, Density: density, UpdatedAt: now,
	}, nil
}

func parseFont(s string) (AppearanceFont, error) {
	switch f := AppearanceFont(s); f {
	case FontAtkinson, FontInter, FontIBMPlexSans, FontSourceSans, FontBarlow:
		return f, nil
	}
	return "", ErrInvalidAppearanceFont
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

var hexColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)

// ParseHexColor accepts "#RRGGBB" in either case and upper-cases it.
func ParseHexColor(s string) (HexColor, error) {
	up := strings.ToUpper(s)
	if !hexColorPattern.MatchString(up) {
		return "", ErrInvalidAppearanceColor
	}
	return HexColor(up), nil
}

// ContrastRatio is the WCAG 2.x contrast ratio between two colours, from 1
// (identical luminance) to 21 (black on white). A colour that is not
// valid #RRGGBB (see ParseHexColor) has no measurable contrast: 1.
func ContrastRatio(a, b HexColor) float64 {
	a, b = HexColor(strings.ToUpper(string(a))), HexColor(strings.ToUpper(string(b)))
	if !hexColorPattern.MatchString(string(a)) || !hexColorPattern.MatchString(string(b)) {
		return 1
	}
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance per WCAG 2.x: linearised sRGB channels weighted
// 0.2126 R + 0.7152 G + 0.0722 B.
func relativeLuminance(c HexColor) float64 {
	s := string(c)
	return 0.2126*linearChannel(s[1:3]) + 0.7152*linearChannel(s[3:5]) + 0.0722*linearChannel(s[5:7])
}

func linearChannel(hex string) float64 {
	v, err := strconv.ParseUint(hex, 16, 8)
	if err != nil { // unreachable: ContrastRatio checked the pattern
		return 0
	}
	c := float64(v) / 255
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}
