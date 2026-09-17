package exceptions_test

import (
	"errors"
	"fmt"
	"testing"

	"sezzlecalculator/backend/exceptions"
)

func TestErrorRemainsTypedWhenWrapped(t *testing.T) {
	original := exceptions.New(exceptions.DivisionByZero, "No se puede dividir entre cero.")
	wrapped := fmt.Errorf("calculation failed: %w", original)
	var got *exceptions.Error
	if !errors.As(wrapped, &got) || got.Code != exceptions.DivisionByZero {
		t.Fatalf("wrapped error lost its code: %v", wrapped)
	}
	if got.Error() != original.Message {
		t.Fatalf("Error() = %q, want %q", got.Error(), original.Message)
	}
}
