package scannerauth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genscannerauth "github.com/adirkuhn/commerceos/api/scannerauth"
	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interface (defined at point of use — ISP) ───────────────────────

type claimPairingCodeUC interface {
	Execute(ctx context.Context, code string) (string, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	claimPairingCode claimPairingCodeUC
}

func NewHandler(claimPairingCode claimPairingCodeUC) *Handler {
	return &Handler{claimPairingCode: claimPairingCode}
}

var _ genscannerauth.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers must mount rg on a router
// group with NO JWT middleware -- see api/scannerauth/generate.go's doc
// comment for why this endpoint is public by design.
func Register(rg gin.IRouter, db *gorm.DB, jwtSecret []byte) {
	repo := repositories.NewScannerPairingCodeRepo(db)
	h := NewHandler(appiam.NewClaimPairingCodeUseCase(db, repo, jwtSecret))
	genscannerauth.RegisterHandlers(rg, h)
}

func (h *Handler) ClaimPairingCode(c *gin.Context) {
	var req genscannerauth.ClaimPairingCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	token, err := h.claimPairingCode.Execute(c.Request.Context(), req.Code)
	if err != nil {
		if errors.Is(err, iamDomain.ErrPairingCodeInvalid) {
			middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
			return
		}
		log.Printf("scannerauth: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	resp := genscannerauth.ClaimPairingCodeResult{Token: token}
	c.JSON(http.StatusOK, genscannerauth.ClaimPairingCodeResponse{Data: &resp})
}
