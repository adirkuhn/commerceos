// Package reviews wires the admin-only (JWT) Reviews moderation API (see
// api/reviews's doc comment) to its use cases.
package reviews

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genreviews "github.com/adirkuhn/commerceos/api/reviews"
	appreviews "github.com/adirkuhn/commerceos/application/reviews"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listReviewsUC interface {
	Execute(ctx context.Context) ([]reviewsDomain.Review, error)
}

type deleteReviewUC interface {
	Execute(ctx context.Context, id string) error
}

type Handler struct {
	listReviews  listReviewsUC
	deleteReview deleteReviewUC
}

func NewHandler(listReviews listReviewsUC, deleteReview deleteReviewUC) *Handler {
	return &Handler{listReviews: listReviews, deleteReview: deleteReview}
}

var _ genreviews.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewReviewRepo(db)
	h := NewHandler(
		appreviews.NewListReviewsUseCase(db, repo),
		appreviews.NewDeleteReviewUseCase(db, repo, repositories.NewOutboxRepository(db)),
	)
	genreviews.RegisterHandlers(rg, h)
}

func (h *Handler) ListReviews(c *gin.Context) {
	reviews, err := h.listReviews.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genreviews.AdminReview, len(reviews))
	for i, r := range reviews {
		out[i] = toGenReview(r)
	}
	middleware.OK(c, out)
}

func (h *Handler) DeleteReview(c *gin.Context, id string) {
	err := h.deleteReview.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenReview(r reviewsDomain.Review) genreviews.AdminReview {
	return genreviews.AdminReview{
		Id: r.ID, SalesChannelId: r.SalesChannelID, ProductId: r.ProductID,
		CustomerName: r.CustomerName, CustomerEmail: r.CustomerEmail, Rating: r.Rating,
		Title: r.Title, Body: r.Body, Verified: r.Verified, CreatedAt: r.CreatedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, reviewsDomain.ErrReviewNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	default:
		log.Printf("reviews: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
