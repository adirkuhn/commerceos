package procurement

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Document upload route handlers ───────────────────────────────────────────

func (h *Handler) AttachDocumentToPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrFileRequired)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	defer f.Close()

	job, err := h.attachDocument.Execute(c.Request.Context(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: id, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

func (h *Handler) RetryDocumentParsingJob(c *gin.Context, id genprocurement.Id) {
	job, err := h.retryDocumentParsingJob.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

func (h *Handler) DeletePurchaseOrderDocument(c *gin.Context, id genprocurement.Id) {
	err := h.deletePurchaseOrderDocument.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

func (h *Handler) GetDocumentParsingJob(c *gin.Context, id genprocurement.Id) {
	job, err := h.getDocumentParsingJob.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusOK, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

func (h *Handler) GetReceivingParsingJob(c *gin.Context, id genprocurement.Id) {
	job, err := h.getReceivingParsingJob.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingParsingJob(*job)
	c.JSON(http.StatusOK, genprocurement.ReceivingParsingJobResponse{Data: &resp})
}

func (h *Handler) CreatePurchaseOrderFromDocument(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrFileRequired)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	defer f.Close()

	var supplierID *string
	if v := c.PostForm("supplier_id"); v != "" {
		supplierID = &v
	}
	var warehouseID *string
	if v := c.PostForm("warehouse_id"); v != "" {
		warehouseID = &v
	}
	var expectedDate *time.Time
	if v := c.PostForm("expected_date"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			expectedDate = &parsed
		}
	}

	job, err := h.createPurchaseOrderFromDoc.Execute(c.Request.Context(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		SupplierID: supplierID, WarehouseID: warehouseID, CurrencyCode: c.PostForm("currency_code"),
		ExpectedDate: expectedDate, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

// ── Receiving route handlers ─────────────────────────────────────────────────

func (h *Handler) UploadInvoice(c *gin.Context, id genprocurement.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrFileRequired)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	defer f.Close()

	job, err := h.uploadInvoice.Execute(c.Request.Context(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: id, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.ReceivingParsingJobResponse{Data: &resp})
}
