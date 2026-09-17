package business

import (
	"math"

	"sezzlecalculator/backend/exceptions"
)

// Operation identifies a supported calculation.
type Operation string

const (
	Add        Operation = "add"
	Subtract   Operation = "subtract"
	Multiply   Operation = "multiply"
	Divide     Operation = "divide"
	Power      Operation = "power"
	SquareRoot Operation = "sqrt"
	Percentage Operation = "percentage"
)

// Service is a stateless calculator, safe to share across concurrent requests.
type Service struct{}

// Calculate applies an operation to finite float64 values.
// SquareRoot ignores b.
// Percentage means a multiplied by b percent.
// Power follows math.Pow's 0^0 = 1 convention.
func (Service) Calculate(operation Operation, a, b float64) (float64, error) {
	if !finite(a) || (operation != SquareRoot && !finite(b)) {
		return 0, exceptions.New(exceptions.InvalidInput, "Los operandos deben ser números finitos.")
	}

	var result float64
	switch operation {
	case Add:
		result = a + b
	case Subtract:
		result = a - b
	case Multiply:
		result = a * b
	case Divide:
		if b == 0 {
			return 0, exceptions.New(exceptions.DivisionByZero, "No se puede dividir entre cero.")
		}
		result = a / b
	case Power:
		if a == 0 && b < 0 {
			return 0, exceptions.New(exceptions.InvalidExponent, "Cero no puede elevarse a un exponente negativo.")
		}
		if a < 0 && b != math.Trunc(b) {
			return 0, exceptions.New(exceptions.InvalidExponent, "Una base negativa requiere un exponente entero para obtener un resultado real.")
		}
		result = math.Pow(a, b)
	case SquareRoot:
		if a < 0 {
			return 0, exceptions.New(exceptions.InvalidSquareRoot, "La raíz cuadrada requiere un número mayor o igual a cero.")
		}
		result = math.Sqrt(a)
	case Percentage:
		product := a * b
		if math.IsInf(product, 0) {
			// Scale first only when necessary, preserving small representable products while avoiding an avoidable intermediate overflow.
			result = (a / 100) * b
		} else {
			result = product / 100
		}
	default:
		return 0, exceptions.New(exceptions.InvalidOperation, "La operación solicitada no es válida.")
	}
	if !finite(result) {
		return 0, exceptions.New(exceptions.NonFiniteResult, "El resultado excede el rango de números finitos admitido.")
	}
	// Normalize negative zero for consistent JSON and display.
	if result == 0 {
		result = 0
	}
	return result, nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
