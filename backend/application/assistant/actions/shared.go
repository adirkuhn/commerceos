// Package actions holds the assistant's proposal actions (see
// application/assistant.ProposalAction): changes the model can propose and
// a person confirms. Prepare only reads -- it checks and prices the
// proposal; Execute delegates to the use case that makes the change and
// records its event.
package actions

//events:delegated CreateQuoteUseCase and ConvertQuoteToSalesOrderUseCase record quotes.quote.created / quotes.quote.converted

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// ErrInvalidArguments: the model's arguments don't match the tool's
// schema. The message goes back to the model so it can correct itself.
var ErrInvalidArguments = errors.New("invalid arguments")

// decode reads the model's arguments strictly: unknown fields are an error.
func decode(args json.RawMessage, v any) error {
	trimmed := bytes.TrimSpace(args)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return fmt.Errorf("%w: arguments are required", ErrInvalidArguments)
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArguments, err)
	}
	return nil
}

func encode(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return b, nil
}

func object(props map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

// discountBasisPoints turns a percentage (12.5) into basis points (1250),
// rounded like the admin's quote editor (Math.round(percent * 100)).
func discountBasisPoints(percent float64) int {
	return int(math.Round(percent * 100))
}

// applyDiscount mirrors frontend/src/pricelist/pricing.ts applyDiscount:
// round(price * (10000 - bp) / 10000), half rounded up -- so a quote the
// assistant prices matches one built in the quote editor to the cent.
func applyDiscount(priceMinor int64, basisPoints int) int64 {
	return int64(math.Round(float64(priceMinor*int64(10000-basisPoints)) / 10000))
}
