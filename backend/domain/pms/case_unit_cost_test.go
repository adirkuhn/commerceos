package pms_test

import (
	"testing"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
)

func TestUnitCostFromCaseMinorUnits(t *testing.T) {
	cases := []struct {
		name         string
		caseCost     int64
		unitsPerCase int64
		want         int64
	}{
		{"exact division", 1000, 20, 50},
		{"rounds half up", 1399, 20, 70}, // 69.95
		{"rounds down below half", 1001, 20, 50},
		{"exactly half rounds up", 30, 20, 2}, // 1.5
		{"single unit case", 1399, 1, 1399},
		{"zero cost", 0, 20, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pmsDomain.UnitCostFromCaseMinorUnits(tc.caseCost, tc.unitsPerCase))
		})
	}
}
