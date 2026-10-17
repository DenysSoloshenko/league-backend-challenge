package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postMatrix(t *testing.T, path, contents string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "matrix.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	newHandler().ServeHTTP(response, request)
	return response
}

func TestInvert(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
	}{
		{"example", "1,2,3\n4,5,6\n7,8,9\n", "1,4,7\n2,5,8\n3,6,9\n"},
		{"single cell", "-42\n", "-42\n"},
		{"two by two", "0,-2\n3,4\n", "0,3\n-2,4\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := postMatrix(t, "/invert", test.input)
			if response.Code != http.StatusOK || response.Body.String() != test.want {
				t.Fatalf("status %d, body %q; want %q", response.Code, response.Body.String(), test.want)
			}
		})
	}
}

func TestEchoStillWorks(t *testing.T) {
	input := "1,2,3\n4,5,6\n7,8,9\n"
	response := postMatrix(t, "/echo", input)
	if response.Code != http.StatusOK || response.Body.String() != input {
		t.Fatalf("status %d, body %q; want %q", response.Code, response.Body.String(), input)
	}
}

func TestFlatten(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
	}{
		{"example", "1,2,3\n4,5,6\n7,8,9\n", "1,2,3,4,5,6,7,8,9"},
		{"single cell", "-42\n", "-42"},
		{"row order and negatives", "0,-2\n3,4\n", "0,-2,3,4"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := postMatrix(t, "/flatten", test.input)
			if response.Code != http.StatusOK || response.Body.String() != test.want {
				t.Fatalf("status %d, body %q; want %q", response.Code, response.Body.String(), test.want)
			}
		})
	}
}

func TestFlattenRejectsInvalidMatrix(t *testing.T) {
	response := postMatrix(t, "/flatten", "1,2\n3\n")
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "matrix must be square") {
		t.Fatalf("status %d, body %q; want square-matrix error", response.Code, response.Body.String())
	}
}

func TestInvertRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name, input, wantError string
	}{
		{"empty", "", "matrix must not be empty"},
		{"non-square", "1,2\n3,4\n5,6\n", "matrix must be square"},
		{"ragged row", "1,2\n3\n", "matrix must be square"},
		{"non-integer", "1,2\n3,x\n", "invalid integer at row 2, column 2"},
		{"malformed CSV", "\"1,2\n", "invalid CSV"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := postMatrix(t, "/invert", test.input)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), test.wantError) {
				t.Fatalf("status %d, body %q; want %q", response.Code, response.Body.String(), test.wantError)
			}
		})
	}
}

func TestInvertRequestValidation(t *testing.T) {
	handler := newHandler()
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/invert", nil))
	if get.Code != http.StatusMethodNotAllowed || get.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("GET: status %d, Allow %q", get.Code, get.Header().Get("Allow"))
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/invert", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing file: status %d", missing.Code)
	}
}
