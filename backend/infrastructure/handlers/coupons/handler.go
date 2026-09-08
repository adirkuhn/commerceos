// Package coupons wires the admin-only (JWT) Coupons API (see
// api/coupons's doc comment) to its use cases.
package coupons

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gencoupons "github.com/adirkuhn/commerceos/api/coupons"
	appCoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listCouponsUC interface {
	Execute(ctx context.Context) ([]couponsDomain.Coupon, error)
}

type createCouponUC interface {
	Execute(ctx context.Context, in appCoupons.CreateCouponInput) (*couponsDomain.Coupon, error)
}

type updateCouponUC interface {
	Execute(ctx context.Context, in appCoupons.UpdateCouponInput) (*couponsDomain.Coupon, error)
}

type deleteCouponUC interface {
	Execute(ctx context.Context, id string) error
}
type listCouponCurrencyAmountsUC interface {
	Execute(ctx context.Context, couponID string) ([]couponsDomain.CouponCurrencyAmount, error)
}
type upsertCouponCurrencyAmountUC interface {
	Execute(ctx context.Context, in appCoupons.UpsertCouponCurrencyAmountInput) (*couponsDomain.CouponCurrencyAmount, error)
}
type deleteCouponCurrencyAmountUC interface {
	Execute(ctx context.Context, couponID, currencyCode string) error
}

type Handler struct {
	listCoupons                listCouponsUC
	createCoupon               createCouponUC
	updateCoupon               updateCouponUC
	deleteCoupon               deleteCouponUC
	listCouponCurrencyAmounts  listCouponCurrencyAmountsUC
	upsertCouponCurrencyAmount upsertCouponCurrencyAmountUC
	deleteCouponCurrencyAmount deleteCouponCurrencyAmountUC
}

func NewHandler(
	listCoupons listCouponsUC, createCoupon createCouponUC, updateCoupon updateCouponUC, deleteCoupon deleteCouponUC,
	listCouponCurrencyAmounts listCouponCurrencyAmountsUC, upsertCouponCurrencyAmount upsertCouponCurrencyAmountUC, deleteCouponCurrencyAmount deleteCouponCurrencyAmountUC,
) *Handler {
	return &Handler{
		listCoupons: listCoupons, createCoupon: createCoupon, updateCoupon: updateCoupon, deleteCoupon: deleteCoupon,
		listCouponCurrencyAmounts: listCouponCurrencyAmounts, upsertCouponCurrencyAmount: upsertCouponCurrencyAmount, deleteCouponCurrencyAmount: deleteCouponCurrencyAmount,
	}
}

var _ gencoupons.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewCouponRepo(db)
	currencyAmountRepo := repositories.NewCouponCurrencyAmountRepo(db)
	h := NewHandler(
		appCoupons.NewListCouponsUseCase(db, repo),
		appCoupons.NewCreateCouponUseCase(db, repo),
		appCoupons.NewUpdateCouponUseCase(db, repo),
		appCoupons.NewDeleteCouponUseCase(db, repo),
		appCoupons.NewListCouponCurrencyAmountsUseCase(db, currencyAmountRepo),
		appCoupons.NewUpsertCouponCurrencyAmountUseCase(db, repo, currencyAmountRepo),
		appCoupons.NewDeleteCouponCurrencyAmountUseCase(db, currencyAmountRepo),
	)
	gencoupons.RegisterHandlers(rg, h)
}

func (h *Handler) ListCoupons(c *gin.Context) {
	coupons, err := h.listCoupons.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]gencoupons.Coupon, len(coupons))
	for i, coupon := range coupons {
		out[i] = toGenCoupon(coupon)
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	var req gencoupons.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	coupon, err := h.createCoupon.Execute(c.Request.Context(), appCoupons.CreateCouponInput{
		Code: req.Code, DiscountType: couponsDomain.DiscountType(req.DiscountType), DiscountValue: req.DiscountValue,
		MinSubtotalMinorUnits: req.MinSubtotalMinorUnits, UsageLimit: req.UsageLimit, ExpiresAt: req.ExpiresAt,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenCoupon(*coupon)})
}

func (h *Handler) UpdateCoupon(c *gin.Context, id string) {
	var req gencoupons.UpdateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	coupon, err := h.updateCoupon.Execute(c.Request.Context(), appCoupons.UpdateCouponInput{
		ID: id, Code: req.Code, DiscountType: couponsDomain.DiscountType(req.DiscountType), DiscountValue: req.DiscountValue,
		MinSubtotalMinorUnits: req.MinSubtotalMinorUnits, UsageLimit: req.UsageLimit, ExpiresAt: req.ExpiresAt, Active: req.Active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCoupon(*coupon))
}

func (h *Handler) DeleteCoupon(c *gin.Context, id string) {
	err := h.deleteCoupon.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListCouponCurrencyAmounts(c *gin.Context, id string) {
	amounts, err := h.listCouponCurrencyAmounts.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]gencoupons.CouponCurrencyAmount, len(amounts))
	for i, a := range amounts {
		out[i] = toGenCouponCurrencyAmount(a)
	}
	middleware.OK(c, out)
}

func (h *Handler) UpsertCouponCurrencyAmount(c *gin.Context, id string) {
	var req gencoupons.UpsertCouponCurrencyAmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	amount, err := h.upsertCouponCurrencyAmount.Execute(c.Request.Context(), appCoupons.UpsertCouponCurrencyAmountInput{
		CouponID: id, CurrencyCode: req.CurrencyCode, AmountMinorUnits: req.AmountMinorUnits,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCouponCurrencyAmount(*amount))
}

func (h *Handler) DeleteCouponCurrencyAmount(c *gin.Context, id string, currencyCode string) {
	err := h.deleteCouponCurrencyAmount.Execute(c.Request.Context(), id, currencyCode)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenCouponCurrencyAmount(a couponsDomain.CouponCurrencyAmount) gencoupons.CouponCurrencyAmount {
	return gencoupons.CouponCurrencyAmount{CouponId: a.CouponID, CurrencyCode: a.CurrencyCode, AmountMinorUnits: a.AmountMinorUnits}
}

func toGenCoupon(c couponsDomain.Coupon) gencoupons.Coupon {
	return gencoupons.Coupon{
		Id: c.ID, Code: c.Code, DiscountType: gencoupons.DiscountType(c.DiscountType), DiscountValue: c.DiscountValue,
		MinSubtotalMinorUnits: c.MinSubtotalMinorUnits, UsageLimit: c.UsageLimit, TimesUsed: c.TimesUsed,
		ExpiresAt: c.ExpiresAt, Active: c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, couponsDomain.ErrCouponCodeRequired),
		errors.Is(err, couponsDomain.ErrCouponCodeTaken),
		errors.Is(err, couponsDomain.ErrInvalidDiscountType),
		errors.Is(err, couponsDomain.ErrInvalidDiscountValue),
		errors.Is(err, couponsDomain.ErrInvalidMinSubtotal),
		errors.Is(err, couponsDomain.ErrInvalidUsageLimit),
		errors.Is(err, fxDomain.ErrInvalidCurrencyCode):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, couponsDomain.ErrCouponNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("coupons: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
