package scannerauth

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"log"
	"net/http"
	"time"

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
	ExecuteByToken(ctx context.Context, token string) (string, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

// claimAttemptWindow bounds the digit-code path (see randomPairingToken vs
// randomSixDigitCode in application/iam/generate_pairing_code.go): a 6-digit
// code has only 1e6 possible values, so this is the actual defense against
// brute-forcing it before its 5-minute TTL expires. Matches that TTL.
const claimAttemptWindow = 5 * time.Minute
const claimAttemptMax = 10

type Handler struct {
	claimPairingCode claimPairingCodeUC
	limiter          *middleware.IPRateLimiter
}

func NewHandler(claimPairingCode claimPairingCodeUC, limiter *middleware.IPRateLimiter) *Handler {
	return &Handler{claimPairingCode: claimPairingCode, limiter: limiter}
}

var _ genscannerauth.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers must mount rg on a router
// group with NO JWT middleware -- see api/scannerauth/generate.go's doc
// comment for why this endpoint is public by design.
func Register(rg gin.IRouter, db *gorm.DB, jwtSecret []byte) {
	repo := repositories.NewScannerPairingCodeRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	limiter := middleware.NewIPRateLimiter(claimAttemptMax, claimAttemptWindow)
	h := NewHandler(appiam.NewClaimPairingCodeUseCase(db, repo, jwtSecret, outboxRepo), limiter)
	genscannerauth.RegisterHandlers(rg, h)
}

// ClaimPairingCode is the digit-code path -- guessable within its keyspace,
// so it's rate-limited per IP. See ClaimPairingCodeByToken for the
// unguessable QR/barcode path, which carries no such limit.
func (h *Handler) ClaimPairingCode(c *gin.Context) {
	if !h.limiter.Allow(c.ClientIP()) {
		middleware.FailErr(c, http.StatusTooManyRequests, "RATE_LIMITED", httperrs.ErrRateLimited)
		return
	}
	var req genscannerauth.ClaimPairingCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	token, err := h.claimPairingCode.Execute(c.Request.Context(), req.Code)
	if err != nil {
		if errors.Is(err, iamDomain.ErrPairingCodeInvalid) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
		log.Printf("scannerauth: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	resp := genscannerauth.ClaimPairingCodeResult{Token: token}
	c.JSON(http.StatusOK, genscannerauth.ClaimPairingCodeResponse{Data: &resp})
}

func (h *Handler) ClaimPairingCodeByToken(c *gin.Context) {
	var req genscannerauth.ClaimPairingCodeByTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	token, err := h.claimPairingCode.ExecuteByToken(c.Request.Context(), req.Token)
	if err != nil {
		if errors.Is(err, iamDomain.ErrPairingCodeInvalid) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
		log.Printf("scannerauth: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	resp := genscannerauth.ClaimPairingCodeResult{Token: token}
	c.JSON(http.StatusOK, genscannerauth.ClaimPairingCodeResponse{Data: &resp})
}
