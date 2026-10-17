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

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", matrixHandler(false))
	mux.HandleFunc("/invert", matrixHandler(true))
	return mux
}

func matrixHandler(invert bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "use POST with a multipart file field named 'file'", http.StatusMethodNotAllowed)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing or invalid file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		matrix, err := readMatrix(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if invert {
			matrix = transpose(matrix)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer := csv.NewWriter(w)
		_ = writer.WriteAll(matrix)
	}
}

// readMatrix accepts a non-empty square CSV matrix containing integers.
func readMatrix(input io.Reader) ([][]string, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	matrix, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(matrix) == 0 {
		return nil, errors.New("matrix must not be empty")
	}
	for rowIndex, row := range matrix {
		if len(row) != len(matrix) {
			return nil, fmt.Errorf("matrix must be square: row %d has %d columns; expected %d", rowIndex+1, len(row), len(matrix))
		}
		for columnIndex, value := range row {
			if _, ok := new(big.Int).SetString(strings.TrimSpace(value), 10); !ok {
				return nil, fmt.Errorf("invalid integer at row %d, column %d: %q", rowIndex+1, columnIndex+1, value)
			}
		}
	}
	return matrix, nil
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
