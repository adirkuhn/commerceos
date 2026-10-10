package suppliers

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrSupplierNotFound = errs.New("SUPPLIER_NOT_FOUND", "supplier not found")
)
