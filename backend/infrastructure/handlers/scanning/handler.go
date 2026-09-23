package scanning

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genscanning "github.com/adirkuhn/commerceos/api/scanning"
	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type generatePairingCodeUC interface {
	Execute(ctx context.Context, in appiam.GeneratePairingCodeInput) (*iamDomain.ScannerPairingCode, error)
}
type getPairingCodeStatusUC interface {
	Execute(ctx context.Context, id string) (*iamDomain.ScannerPairingCode, error)
}
type refreshScannerSessionUC interface {
	Execute(userID, tenantID, role string) (string, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	generatePairingCode   generatePairingCodeUC
	getPairingCodeStatus  getPairingCodeStatusUC
	refreshScannerSession refreshScannerSessionUC
}

func NewHandler(generatePairingCode generatePairingCodeUC, getPairingCodeStatus getPairingCodeStatusUC, refreshScannerSession refreshScannerSessionUC) *Handler {
	return &Handler{generatePairingCode: generatePairingCode, getPairingCodeStatus: getPairingCodeStatus, refreshScannerSession: refreshScannerSession}
}

var _ genscanning.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then mounts
// the generated routes. No role guard beyond the JWT middleware already
// applied to the router group this is called with -- see
// api/scanning/generate.go's doc comment for why this is deliberately
// separate from api/iam's admin-only Groups module.
func Register(rg gin.IRouter, db *gorm.DB, jwtSecret []byte) {
	repo := repositories.NewScannerPairingCodeRepo(db)
	h := NewHandler(
		appiam.NewGeneratePairingCodeUseCase(db, repo),
		appiam.NewGetPairingCodeStatusUseCase(db, repo),
		appiam.NewRefreshScannerSessionUseCase(jwtSecret),
	)
	genscanning.RegisterHandlers(rg, h)
}

func (h *Handler) GeneratePairingCode(c *gin.Context) {
	claims := middleware.ClaimsFromGin(c)
	if claims == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "no claims in context")
		return
	}
	pairing, err := h.generatePairingCode.Execute(c.Request.Context(), appiam.GeneratePairingCodeInput{
		UserID: claims.UserID, Role: string(claims.Role),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPairingCode(*pairing)
	c.JSON(http.StatusCreated, genscanning.ScannerPairingCodeResponse{Data: &resp})
}

func (h *Handler) GetPairingCodeStatus(c *gin.Context, id genscanning.Id) {
	pairing, err := h.getPairingCodeStatus.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPairingCode(*pairing)
	c.JSON(http.StatusOK, genscanning.ScannerPairingCodeResponse{Data: &resp})
}

func (h *Handler) RefreshScannerSession(c *gin.Context) {
	claims := middleware.ClaimsFromGin(c)
	if claims == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "no claims in context")
		return
	}
	token, err := h.refreshScannerSession.Execute(claims.UserID, claims.TenantID, string(claims.Role))
	if err != nil {
		log.Printf("scanning: refresh scanner session: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	resp := genscanning.RefreshScannerSessionResult{Token: token}
	c.JSON(http.StatusOK, genscanning.RefreshScannerSessionResponse{Data: &resp})
}

func toGenPairingCode(p iamDomain.ScannerPairingCode) genscanning.ScannerPairingCode {
	return genscanning.ScannerPairingCode{
		Id: p.ID, TenantId: p.TenantID, Code: p.Code, Token: p.Token, Status: genscanning.ScannerPairingCodeStatus(p.Status),
		ExpiresAt: p.ExpiresAt, CreatedAt: p.CreatedAt, ClaimedAt: p.ClaimedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, iamDomain.ErrPairingCodeNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("scanning: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
