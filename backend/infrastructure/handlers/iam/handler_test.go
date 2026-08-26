package iam_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	iamhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/iam"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const testSecret = "handler-test-secret"

// ── Stub use cases ────────────────────────────────────────────────────────────

type stubCreateGroup struct {
	out *iamDomain.Group
	err error
}

func (s *stubCreateGroup) Execute(_ context.Context, _ appiam.CreateGroupInput) (*iamDomain.Group, error) {
	return s.out, s.err
}

type stubUpdateGroup struct {
	out *iamDomain.Group
	err error
}

func (s *stubUpdateGroup) Execute(_ context.Context, _ appiam.UpdateGroupInput) (*iamDomain.Group, error) {
	return s.out, s.err
}

type stubDeleteGroup struct{ err error }

func (s *stubDeleteGroup) Execute(_ context.Context, _ string) error { return s.err }

type stubGetGroup struct {
	out *iamDomain.Group
	err error
}

func (s *stubGetGroup) Execute(_ context.Context, _ string) (*iamDomain.Group, error) {
	return s.out, s.err
}

type stubListGroups struct {
	out []iamDomain.Group
	err error
}

func (s *stubListGroups) Execute(_ context.Context) ([]iamDomain.Group, error) {
	return s.out, s.err
}

type stubAddMember struct{ err error }

func (s *stubAddMember) Execute(_ context.Context, _ appiam.AddUserToGroupInput) error { return s.err }

type stubRemoveMember struct{ err error }

func (s *stubRemoveMember) Execute(_ context.Context, _ appiam.RemoveUserFromGroupInput) error {
	return s.err
}

type stubListMembers struct {
	out []string
	err error
}

func (s *stubListMembers) Execute(_ context.Context, _ string) ([]string, error) {
	return s.out, s.err
}

type stubAssignPermission struct{ err error }

func (s *stubAssignPermission) Execute(_ context.Context, _ appiam.AssignPermissionInput) error {
	return s.err
}

type stubRevokePermission struct{ err error }

func (s *stubRevokePermission) Execute(_ context.Context, _ appiam.RevokePermissionInput) error {
	return s.err
}

type stubListGroupPermissions struct {
	out []iamDomain.Rule
	err error
}

func (s *stubListGroupPermissions) Execute(_ context.Context, _ string) ([]iamDomain.Rule, error) {
	return s.out, s.err
}

type stubUserEffectivePermissions struct {
	out []iamDomain.Rule
	err error
}

func (s *stubUserEffectivePermissions) Execute(_ context.Context, _ string) ([]iamDomain.Rule, error) {
	return s.out, s.err
}

type stubListUserActivities struct {
	out []omsDomain.SalesOrderActivity
	err error
}

func (s *stubListUserActivities) Execute(_ context.Context, _ string) ([]omsDomain.SalesOrderActivity, error) {
	return s.out, s.err
}

// ── Test helpers ──────────────────────────────────────────────────────────────

func adminToken(t *testing.T) string {
	t.Helper()
	claims := middleware.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Role:     middleware.RoleCustomerOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return s
}

func newTestRouter(h *iamhandler.Handler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.Use(middleware.ErrorHandler())
	v1 := r.Group("/api/v1")
	iamhandler.RegisterHandler(v1, h)
	return r
}

func do(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sampleGroup() *iamDomain.Group {
	return &iamDomain.Group{
		ID:          "group-uuid-1",
		TenantID:    "tenant-1",
		Name:        "Editors",
		Description: "Can edit products",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ── Tests: groups CRUD ────────────────────────────────────────────────────────

func TestCreateGroup_201(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{out: sampleGroup()},
		&stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups",
		map[string]string{"name": "Editors", "description": "Can edit"})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateGroup_422_missing_name(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups",
		map[string]string{"description": "no name"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateGroup_409_name_taken(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{err: iamDomain.ErrGroupNameTaken},
		&stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups",
		map[string]string{"name": "Editors"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetGroup_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{},
		&stubGetGroup{out: sampleGroup()},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/groups/group-uuid-1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetGroup_404(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{},
		&stubGetGroup{err: iamDomain.ErrGroupNotFound},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/groups/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListGroups_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{out: []iamDomain.Group{*sampleGroup()}},
		&stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/groups", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateGroup_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{},
		&stubUpdateGroup{out: sampleGroup()},
		&stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPatch, "/api/v1/iam/groups/group-uuid-1",
		map[string]string{"name": "Editors v2"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteGroup_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/group-uuid-1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteGroup_404(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{},
		&stubDeleteGroup{err: iamDomain.ErrGroupNotFound},
		&stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Tests: members ────────────────────────────────────────────────────────────

func TestAddMember_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups/g1/members",
		map[string]string{"user_id": "user-42"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddMember_409_already_member(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{},
		&stubAddMember{err: iamDomain.ErrAlreadyMember},
		&stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups/g1/members",
		map[string]string{"user_id": "user-42"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRemoveMember_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/g1/members/user-42", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRemoveMember_422_not_member(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{},
		&stubRemoveMember{err: iamDomain.ErrNotMember},
		&stubListMembers{}, &stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/g1/members/user-99", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListMembers_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{},
		&stubListMembers{out: []string{"user-1", "user-2"}},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/groups/g1/members", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Tests: permissions ────────────────────────────────────────────────────────

func TestAssignPermission_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups/g1/permissions",
		map[string]string{"rule": "pms.product.create"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAssignPermission_422_invalid_rule(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{err: iamDomain.ErrPermissionInvalid},
		&stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodPost, "/api/v1/iam/groups/g1/permissions",
		map[string]string{"rule": "bad-rule"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRevokePermission_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/g1/permissions/pms.product.create", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRevokePermission_404_not_assigned(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{},
		&stubRevokePermission{err: iamDomain.ErrPermissionNotAssigned},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodDelete, "/api/v1/iam/groups/g1/permissions/pms.product.create", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListGroupPermissions_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{out: []iamDomain.Rule{"pms.product.create"}},
		&stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/groups/g1/permissions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserEffectivePermissions_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{},
		&stubUserEffectivePermissions{out: []iamDomain.Rule{"pms.product.create", "oms.order.read"}},
		&stubListUserActivities{},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/users/user-1/permissions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListUserActivities_200(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{out: []omsDomain.SalesOrderActivity{
			{ID: "a-1", SalesOrderID: "so-1", UserID: "user-1", ActivityType: "myparcel_shipment_created", Summary: "Shipment created", Metadata: []byte(`{}`)},
		}},
	)
	w := do(t, newTestRouter(h), http.MethodGet, "/api/v1/iam/users/user-1/activities", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "myparcel_shipment_created")
}

// ── Tests: auth ───────────────────────────────────────────────────────────────

func TestIAM_401_no_token(t *testing.T) {
	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	r := newTestRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/iam/groups", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIAM_403_insufficient_role(t *testing.T) {
	claims := middleware.Claims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Role:     middleware.RoleCustomerUser,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)

	h := iamhandler.NewHandler(
		&stubCreateGroup{}, &stubUpdateGroup{}, &stubDeleteGroup{}, &stubGetGroup{},
		&stubListGroups{}, &stubAddMember{}, &stubRemoveMember{}, &stubListMembers{},
		&stubAssignPermission{}, &stubRevokePermission{},
		&stubListGroupPermissions{}, &stubUserEffectivePermissions{},
		&stubListUserActivities{},
	)
	r := newTestRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/iam/groups", nil)
	req.Header.Set("Authorization", "Bearer "+s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
