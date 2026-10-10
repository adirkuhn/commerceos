package storefront

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) ListStorefrontReviews(c *gin.Context, id string, params genstorefront.ListStorefrontReviewsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	in := appstorefront.ListReviewsInput{SalesChannelID: *resolved.SalesChannelID, ProductID: id}
	if params.PerPage != nil {
		in.Limit = *params.PerPage
	}
	if params.Page != nil && in.Limit > 0 {
		in.Offset = (*params.Page - 1) * in.Limit
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.listReviews.Execute(ctx, in)
	if err := mapError(c, err); err != nil {
		return
	}
	reviews := make([]genstorefront.Review, len(result.Reviews))
	for i, rv := range result.Reviews {
		reviews[i] = toGenReview(rv)
	}
	middleware.OK(c, genstorefront.ReviewList{
		Reviews: reviews,
		Summary: genstorefront.ReviewSummary{Average: result.Summary.Average, Count: result.Summary.Count},
	})
}

func (h *Handler) SubmitStorefrontReview(c *gin.Context, id string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	review, err := h.submitReview.Execute(ctx, appstorefront.SubmitReviewInput{
		SalesChannelID: *resolved.SalesChannelID, ProductID: id,
		CustomerName: req.CustomerName, CustomerEmail: string(req.CustomerEmail), Rating: req.Rating,
		Title: strPtrOr(req.Title), Body: req.Body,
	})
	if errors.Is(err, storefrontDomain.ErrProductNotAvailable) {
		middleware.Fail(c, http.StatusNotFound, "PRODUCT_NOT_AVAILABLE", storefrontDomain.ErrProductNotAvailable.Error())
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenReview(*review)})
}

func toGenReview(rv reviewsDomain.Review) genstorefront.Review {
	return genstorefront.Review{
		Id: rv.ID, CustomerName: rv.CustomerName, Rating: rv.Rating, Title: rv.Title, Body: rv.Body,
		Verified: rv.Verified, CreatedAt: rv.CreatedAt,
	}
}
