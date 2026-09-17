package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"sezzlecalculator/backend/business"
	"sezzlecalculator/backend/exceptions"
)

const MaxBodyBytes int64 = 4096

// Calculator is the business boundary needed by the presentation layer.
type Calculator interface {
	Calculate(business.Operation, float64, float64) (float64, error)
}

type handler struct {
	calculator Calculator
	staticDir  string
}

var absolutePath = filepath.Abs
var evalSymlinks = filepath.EvalSymlinks

// NewHandler builds the API and optionally serves a compiled frontend.
func NewHandler(service Calculator, staticDir string) (http.Handler, error) {
	h := &handler{calculator: service}
	if staticDir != "" {
		absolute, err := absolutePath(staticDir)
		if err != nil {
			return nil, err
		}
		h.staticDir, err = evalSymlinks(absolute)
		if err != nil {
			return nil, err
		}
		file, err := h.openStatic("index.html")
		if err != nil {
			return nil, err
		}
		file.Close()
	}
	return h, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("panic while handling request", "method", r.Method, "path", r.URL.Path, "panic", recovered)
			writeError(w, exceptions.New(exceptions.InternalError, "Ocurrió un error interno al procesar la solicitud."))
		}
	}()

	if r.URL.Path == "/api/health" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	operation, exists := operations[r.URL.Path]
	if exists {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		h.calculate(w, r, operation)
		return
	}
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || h.staticDir == "" {
		writeError(w, exceptions.New(exceptions.NotFound, "La ruta solicitada no existe."))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	h.serveStatic(w, r)
}

var operations = map[string]business.Operation{
	"/api/v1/add":        business.Add,
	"/api/v1/subtract":   business.Subtract,
	"/api/v1/multiply":   business.Multiply,
	"/api/v1/divide":     business.Divide,
	"/api/v1/power":      business.Power,
	"/api/v1/sqrt":       business.SquareRoot,
	"/api/v1/percentage": business.Percentage,
}

func (h *handler) calculate(w http.ResponseWriter, r *http.Request, operation business.Operation) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, exceptions.New(exceptions.UnsupportedMediaType, "Utiliza Content-Type: application/json."))
		return
	}
	if r.ContentLength > MaxBodyBytes {
		writeError(w, exceptions.New(exceptions.PayloadTooLarge, "El cuerpo de la solicitud supera el límite de 4096 bytes."))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var a, b *float64
	if operation == business.SquareRoot {
		var input struct {
			A *float64 `json:"a"`
		}
		err = decoder.Decode(&input)
		a = input.A
	} else {
		var input struct {
			A *float64 `json:"a"`
			B *float64 `json:"b"`
		}
		err = decoder.Decode(&input)
		a, b = input.A, input.B
	}
	if err != nil {
		writeDecodeError(w, err)
		return
	}
	// A single JSON document is required
	if err = decoder.Decode(new(any)); err != io.EOF {
		writeDecodeError(w, err)
		return
	}
	if a == nil || (operation != business.SquareRoot && b == nil) {
		writeError(w, exceptions.New(exceptions.InvalidInput, "Incluye todos los operandos requeridos como números; no se admite null."))
		return
	}
	var second float64
	if b != nil {
		second = *b
	}
	result, err := h.calculator.Calculate(operation, *a, second)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Operation business.Operation `json:"operation"`
		Result    float64            `json:"result"`
	}{operation, result})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, exceptions.New(exceptions.PayloadTooLarge, "El cuerpo de la solicitud supera el límite de 4096 bytes."))
		return
	}
	writeError(w, exceptions.New(exceptions.InvalidJSON, "Envía un único objeto JSON válido, con los campos esperados y valores numéricos."))
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeError(w, exceptions.New(exceptions.MethodNotAllowed, "El método HTTP no está permitido para esta ruta."))
}

func writeError(w http.ResponseWriter, err error) {
	var appErr *exceptions.Error
	if !errors.As(err, &appErr) {
		appErr = exceptions.New(exceptions.InternalError, "Ocurrió un error interno al procesar la solicitud.")
	}
	status := http.StatusInternalServerError
	switch appErr.Code {
	case exceptions.InvalidInput, exceptions.InvalidJSON, exceptions.InvalidOperation:
		status = http.StatusBadRequest
	case exceptions.DivisionByZero, exceptions.InvalidSquareRoot, exceptions.InvalidExponent, exceptions.NonFiniteResult:
		status = http.StatusUnprocessableEntity
	case exceptions.UnsupportedMediaType:
		status = http.StatusUnsupportedMediaType
	case exceptions.PayloadTooLarge:
		status = http.StatusRequestEntityTooLarge
	case exceptions.NotFound:
		status = http.StatusNotFound
	case exceptions.MethodNotAllowed:
		status = http.StatusMethodNotAllowed
	}
	writeJSON(w, status, struct {
		Error *exceptions.Error `json:"error"`
	}{appErr})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	// Marshal before committing headers so serialization failures remain JSON.
	body, err := json.Marshal(payload)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"INTERNAL_ERROR","message":"No se pudo serializar la respuesta."}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(append(body, '\n'))
}
