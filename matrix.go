package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
)

const maxRequestBytes = 10 << 20 // Includes multipart overhead.

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", matrixHandler("echo"))
	mux.HandleFunc("/invert", matrixHandler("invert"))
	mux.HandleFunc("/flatten", matrixHandler("flatten"))
	mux.HandleFunc("/sum", matrixHandler("sum"))
	mux.HandleFunc("/multiply", matrixHandler("multiply"))
	return mux
}

func matrixHandler(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "use POST with a multipart file field named 'file'", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		file, _, err := r.FormFile("file")
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "request exceeds 10 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "missing or invalid file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		matrix, err := readMatrix(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var output string
		switch operation {
		case "echo":
			output = encodeMatrix(matrix)
		case "invert":
			output = encodeMatrix(transpose(matrix))
		case "flatten":
			var values strings.Builder
			for _, row := range matrix {
				for _, value := range row {
					if values.Len() > 0 {
						values.WriteByte(',')
					}
					values.WriteString(value)
				}
			}
			output = values.String()
		case "sum", "multiply":
			output = calculate(matrix, operation == "multiply")
		default:
			http.Error(w, "unsupported operation", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, output)
	}
}

// readMatrix accepts a non-empty square CSV matrix containing integers.
func readMatrix(input io.Reader) ([][]string, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	cells, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(cells) == 0 {
		return nil, errors.New("matrix must not be empty")
	}
	for rowIndex, row := range cells {
		if len(row) != len(cells) {
			return nil, fmt.Errorf("matrix must be square: row %d has %d columns; expected %d", rowIndex+1, len(row), len(cells))
		}
		for columnIndex, value := range row {
			value = strings.TrimSpace(value)
			if !isDecimalInteger(value) {
				return nil, fmt.Errorf("invalid integer at row %d, column %d: %q", rowIndex+1, columnIndex+1, value)
			}
			row[columnIndex] = value
		}
	}
	return cells, nil
}

func isDecimalInteger(value string) bool {
	if len(value) > 0 && (value[0] == '+' || value[0] == '-') {
		value = value[1:]
	}
	if len(value) == 0 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func encodeMatrix(cells [][]string) string {
	var output strings.Builder
	for _, row := range cells {
		for column, value := range row {
			if column > 0 {
				output.WriteByte(',')
			}
			output.WriteString(value)
		}
		output.WriteByte('\n')
	}
	return output.String()
}

func calculate(cells [][]string, multiply bool) string {
	result := big.NewInt(0)
	if multiply {
		result.SetInt64(1)
	}
	var integer big.Int
	for _, row := range cells {
		for _, value := range row {
			integer.SetString(value, 10) // Decimal syntax is validated by readMatrix.
			if multiply {
				result.Mul(result, &integer)
			} else {
				result.Add(result, &integer)
			}
		}
	}
	return result.String()
}

func transpose(matrix [][]string) [][]string {
	result := make([][]string, len(matrix))
	for row := range result {
		result[row] = make([]string, len(matrix))
		for column := range result[row] {
			result[row][column] = matrix[column][row]
		}
	}
	return result
}
