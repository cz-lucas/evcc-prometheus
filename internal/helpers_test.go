package evccprometheus

import (
	"testing"
)

func TestRoundFloatValidPrecision(t *testing.T) {
	tests := []struct {
		val       float64
		precision uint
		want      float64
	}{
		{val: 1.2345, precision: 0, want: 1},
		{val: 1.5678, precision: 1, want: 1.6},
		{val: 0.1234, precision: 2, want: 0.12},
		{val: 0.1267, precision: 2, want: 0.13},
		{val: 1.2345, precision: 3, want: 1.235},
		{val: 1.23, precision: 3, want: 1.23},
	}

	for _, tt := range tests {
		got := roundFloat(tt.val, tt.precision)
		if got != tt.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tt.val, tt.precision, got, tt.want)
		}
	}
}
