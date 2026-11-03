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
	"strings"
	"sync"
)

// Coded is a domain error with a stable code.
type Coded struct {
	code    string
	message string
	// showCause: the text wrapped after this error (e.g. an upstream
	// provider's own reason) is useful to the reader -- see DetailOf.
	showCause bool
	// parent makes errors.Is(e, parent) true -- see NewChild.
	parent *Coded
}

func (e *Coded) Error() string { return e.message }

// Unwrap exposes the parent of a NewChild error to errors.Is.
func (e *Coded) Unwrap() error {
	if e.parent == nil {
		return nil
	}
	return e.parent
}

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

// NewChild is a more specific kind of parent: errors.Is(child, parent) is
// true, while clients get the child's own code and sentence.
func NewChild(parent *Coded, code, message string) *Coded {
	c := New(code, message)
	c.parent = parent
	return c
}

// NewWithCause is New for errors whose wrapped detail the reader needs,
// e.g. a provider's reason in fmt.Errorf("%w: 401 unauthorized", err).
func NewWithCause(code, message string) *Coded {
	c := New(code, message)
	c.showCause = true
	return c
}

// NewChildWithCause is NewChild for a child whose wrapped detail the
// reader needs (see NewWithCause).
func NewChildWithCause(parent *Coded, code, message string) *Coded {
	c := NewChild(parent, code, message)
	c.showCause = true
	return c
}

// DetailOf returns the detail wrapped after a NewWithCause error in err's
// chain ("401 unauthorized" above), if there is one.
func DetailOf(err error) (string, bool) {
	c := mostSpecific(err)
	if c == nil || !c.showCause {
		return "", false
	}
	full := err.Error()
	i := strings.Index(full, c.message+": ")
	if i < 0 {
		return "", false
	}
	detail := full[i+len(c.message)+2:]
	return detail, detail != ""
}

// CodeOf finds the coded error in err's chain: its code and its own
// sentence (without the call-site prefixes wrapping added).
//
// When the chain also holds a child of that error (a caller wrapped a
// generic parent around a cause that already says more, e.g.
// "%w: %w", ErrUpstreamFailed, adapterErr), the child wins.
func CodeOf(err error) (code, message string, ok bool) {
	c := mostSpecific(err)
	if c == nil {
		return "", "", false
	}
	return c.code, c.message, true
}

// mostSpecific is the first coded error in err's chain, refined to a
// descendant of it found anywhere in the same chain.
func mostSpecific(err error) *Coded {
	var first *Coded
	if !errors.As(err, &first) {
		return nil
	}
	best := first
	walk(err, func(e error) {
		if c, ok := e.(*Coded); ok && c.descendsFrom(best) {
			best = c
		}
	})
	return best
}

func (e *Coded) descendsFrom(ancestor *Coded) bool {
	for p := e.parent; p != nil; p = p.parent {
		if p == ancestor {
			return true
		}
	}
	return false
}

// walk visits every error in err's tree, depth-first.
func walk(err error, visit func(error)) {
	if err == nil {
		return
	}
	visit(err)
	switch u := err.(type) {
	case interface{ Unwrap() []error }:
		for _, e := range u.Unwrap() {
			walk(e, visit)
		}
	case interface{ Unwrap() error }:
		walk(u.Unwrap(), visit)
	}
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
