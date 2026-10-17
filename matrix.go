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

type matrixData struct {
	cells    [][]string
	integers [][]*big.Int
}

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
			output, err = encodeMatrix(matrix.cells)
		case "invert":
			output, err = encodeMatrix(transpose(matrix.cells))
		case "flatten":
			values := make([]string, 0, len(matrix.cells)*len(matrix.cells))
			for _, row := range matrix.cells {
				values = append(values, row...)
			}
			output = strings.Join(values, ",")
		case "sum", "multiply":
			output = calculate(matrix.integers, operation == "multiply")
		default:
			http.Error(w, "unsupported operation", http.StatusInternalServerError)
			return
		}
		if err != nil {
			http.Error(w, "cannot encode matrix: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, output)
	}
}

// readMatrix accepts a non-empty square CSV matrix containing integers.
func readMatrix(input io.Reader) (matrixData, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	cells, err := reader.ReadAll()
	if err != nil {
		return matrixData{}, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(cells) == 0 {
		return matrixData{}, errors.New("matrix must not be empty")
	}
	integers := make([][]*big.Int, len(cells))
	for rowIndex, row := range cells {
		if len(row) != len(cells) {
			return matrixData{}, fmt.Errorf("matrix must be square: row %d has %d columns; expected %d", rowIndex+1, len(row), len(cells))
		}
		integers[rowIndex] = make([]*big.Int, len(row))
		for columnIndex, value := range row {
			integer, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
			if !ok {
				return matrixData{}, fmt.Errorf("invalid integer at row %d, column %d: %q", rowIndex+1, columnIndex+1, value)
			}
			integers[rowIndex][columnIndex] = integer
		}
	}
	return matrixData{cells: cells, integers: integers}, nil
}

func encodeMatrix(cells [][]string) (string, error) {
	var output strings.Builder
	writer := csv.NewWriter(&output)
	if err := writer.WriteAll(cells); err != nil {
		return "", err
	}
	return output.String(), nil
}

func calculate(integers [][]*big.Int, multiply bool) string {
	result := big.NewInt(0)
	if multiply {
		result.SetInt64(1)
	}
	for _, row := range integers {
		for _, integer := range row {
			if multiply {
				result.Mul(result, integer)
			} else {
				result.Add(result, integer)
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
