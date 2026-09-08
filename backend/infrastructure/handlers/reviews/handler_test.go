package reviews_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genreviews "github.com/adirkuhn/commerceos/api/reviews"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	reviewshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/reviews"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListReviews struct {
	out []reviewsDomain.Review
	err error
}

func (s *stubListReviews) Execute(_ context.Context) ([]reviewsDomain.Review, error) {
	return s.out, s.err
}

type stubDeleteReview struct {
	err   error
	gotID string
}

func (s *stubDeleteReview) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newTestHandler(list *stubListReviews, deleteReview *stubDeleteReview) *reviewshandler.Handler {
	return reviewshandler.NewHandler(list, deleteReview)
}

func newTestRouter(h *reviewshandler.Handler) *gin.Engine {
	r := gin.New()
	genreviews.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestListReviewsHandler_returnsTheList(t *testing.T) {
	list := &stubListReviews{out: []reviewsDomain.Review{
		{ID: "review-1", CustomerName: "Ada", Rating: 5, Body: "Great", CreatedAt: time.Now()},
	}}
	h := newTestHandler(list, &stubDeleteReview{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reviews")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Ada")
}

func TestDeleteReviewHandler_success(t *testing.T) {
	deleteReview := &stubDeleteReview{}
	h := newTestHandler(&stubListReviews{}, deleteReview)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/reviews/review-1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "review-1", deleteReview.gotID)
}

func TestDeleteReviewHandler_notFound(t *testing.T) {
	deleteReview := &stubDeleteReview{err: reviewsDomain.ErrReviewNotFound}
	h := newTestHandler(&stubListReviews{}, deleteReview)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/reviews/missing")

	require.Equal(t, http.StatusNotFound, w.Code)
}
