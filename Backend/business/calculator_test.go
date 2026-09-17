package business_test

import (
	"errors"
	"math"
	"testing"

	"sezzlecalculator/backend/business"
	"sezzlecalculator/backend/exceptions"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name string
		op   business.Operation
		a, b float64
		want float64
	}{
		{"sum", business.Add, 2, 3, 5},
		{"negative sum", business.Add, -7, 2, -5},
		{"decimal sum", business.Add, 0.1, 0.2, 0.3},
		{"zero is valid", business.Add, 0, 0, 0},
		{"difference", business.Subtract, 3, 8, -5},
		{"product", business.Multiply, -2.5, 4, -10},
		{"division", business.Divide, 7, 2, 3.5},
		{"zero numerator", business.Divide, 0, 2, 0},
		{"power", business.Power, 2, 10, 1024},
		{"negative power", business.Power, 2, -3, 0.125},
		{"negative base integer power", business.Power, -2, 3, -8},
		{"fractional power", business.Power, 9, 0.5, 3},
		{"zero power zero convention", business.Power, 0, 0, 1},
		{"square root", business.SquareRoot, 81, 0, 9},
		{"square root zero", business.SquareRoot, 0, 0, 0},
		{"square root ignores second operand", business.SquareRoot, 4, math.NaN(), 2},
		{"percentage", business.Percentage, 200, 15, 30},
		{"negative percentage", business.Percentage, -200, 15, -30},
		{"percentage zero", business.Percentage, 200, 0, 0},
		{"percentage avoids intermediate overflow", business.Percentage, math.MaxFloat64, 100, math.MaxFloat64},
		{"percentage preserves small values", business.Percentage, math.SmallestNonzeroFloat64, 100, math.SmallestNonzeroFloat64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := (business.Service{}).Calculate(tt.op, tt.a, tt.b)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			if got != tt.want && math.Abs(got-tt.want) > math.Abs(tt.want)*1e-14 {
				t.Errorf("Calculate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateErrors(t *testing.T) {
	tests := []struct {
		name string
		op   business.Operation
		a, b float64
		code exceptions.Code
	}{
		{"division by zero", business.Divide, 1, 0, exceptions.DivisionByZero},
		{"division by negative zero", business.Divide, 1, math.Copysign(0, -1), exceptions.DivisionByZero},
		{"negative root", business.SquareRoot, -1, 0, exceptions.InvalidSquareRoot},
		{"negative fractional power", business.Power, -2, 0.5, exceptions.InvalidExponent},
		{"zero negative power", business.Power, 0, -1, exceptions.InvalidExponent},
		{"NaN first operand", business.Add, math.NaN(), 2, exceptions.InvalidInput},
		{"infinite first operand", business.Add, math.Inf(1), 2, exceptions.InvalidInput},
		{"NaN second operand", business.Add, 2, math.NaN(), exceptions.InvalidInput},
		{"infinite second operand", business.Divide, 2, math.Inf(-1), exceptions.InvalidInput},
		{"sum overflow", business.Add, math.MaxFloat64, math.MaxFloat64, exceptions.NonFiniteResult},
		{"difference overflow", business.Subtract, -math.MaxFloat64, math.MaxFloat64, exceptions.NonFiniteResult},
		{"product overflow", business.Multiply, math.MaxFloat64, 2, exceptions.NonFiniteResult},
		{"division overflow", business.Divide, math.MaxFloat64, 0.5, exceptions.NonFiniteResult},
		{"power overflow", business.Power, 10, 400, exceptions.NonFiniteResult},
		{"percentage overflow", business.Percentage, math.MaxFloat64, 200, exceptions.NonFiniteResult},
		{"unknown operation", business.Operation("unknown"), 1, 2, exceptions.InvalidOperation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := (business.Service{}).Calculate(tt.op, tt.a, tt.b)
			var appErr *exceptions.Error
			if !errors.As(err, &appErr) || appErr.Code != tt.code {
				t.Fatalf("Calculate() error = %v, want code %s", err, tt.code)
			}
			if appErr.Message == "" {
				t.Fatal("application errors must include a message")
			}
		})
	}
}

func TestNegativeZeroIsNormalized(t *testing.T) {
	got, err := (business.Service{}).Calculate(business.Multiply, 0, -1)
	if err != nil || math.Signbit(got) {
		t.Fatalf("Calculate() = %v, %v; want positive zero", got, err)
	}
}
