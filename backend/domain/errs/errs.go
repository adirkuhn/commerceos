// Package errs gives domain errors a stable code, so every client can show
// them in the reader's language (admin errors.json, storefront shop.json)
// instead of the English sentence. A coded error is still a plain sentinel:
// compare it with errors.Is as before.
//
//	var ErrProductNotFound = errs.New("PRODUCT_NOT_FOUND", "product not found")
package errs

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"sync"
)

// Coded is a domain error with a stable code.
type Coded struct {
	code    string
	message string
}

func (e *Coded) Error() string { return e.message }

// Code is the stable, client-facing identifier, e.g. PRODUCT_NOT_FOUND.
func (e *Coded) Code() string { return e.code }

var (
	codePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*[A-Z0-9]$`)
	mu          sync.Mutex
	registry    = map[string]bool{}
)

// New registers a coded sentinel error. Codes are UPPER_SNAKE_CASE and
// unique across the whole backend; a bad or reused code panics at start-up.
func New(code, message string) *Coded {
	if !codePattern.MatchString(code) {
		panic(fmt.Sprintf("errs: code %q must be UPPER_SNAKE_CASE", code))
	}
	mu.Lock()
	defer mu.Unlock()
	if registry[code] {
		panic(fmt.Sprintf("errs: code %q registered twice", code))
	}
	registry[code] = true
	return &Coded{code: code, message: message}
}

// CodeOf finds the coded error in err's chain: its code and its own
// sentence (without the call-site prefixes wrapping added).
func CodeOf(err error) (code, message string, ok bool) {
	var c *Coded
	if !errors.As(err, &c) {
		return "", "", false
	}
	return c.code, c.message, true
}

// Codes lists every registered code, sorted -- e.g. for the translation
// parity test.
func Codes() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, 0, len(registry))
	for c := range registry {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
