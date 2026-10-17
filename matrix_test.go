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

func TestEndpointsNormalizeWhitespace(t *testing.T) {
	input := "\"\n +001 \r\n\", 2 \r\n\" 3 \",\"-4\n\"\r\n"
	for _, test := range []struct{ path, want string }{
		{"/echo", "+001,2\n3,-4\n"},
		{"/invert", "+001,3\n2,-4\n"},
		{"/flatten", "+001,2,3,-4"},
		{"/sum", "2"},
		{"/multiply", "-24"},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := postMatrix(t, test.path, input)
			if response.Code != http.StatusOK || response.Body.String() != test.want {
				t.Fatalf("status %d, body %q; want %q", response.Code, response.Body.String(), test.want)
			}
		})
	}
}

func TestSumAndMultiply(t *testing.T) {
	for _, test := range []struct {
		name, input, sum, product string
	}{
		{"example", "1,2,3\n4,5,6\n7,8,9\n", "45", "362880"},
		{"negative values", "-1,2\n3,4\n", "8", "-24"},
		{"zero", "-1,0\n3,4\n", "6", "0"},
		{"one large integer", "9223372036854775808\n", "9223372036854775808", "9223372036854775808"},
		{"large values across cells", "9223372036854775808,2\n3,4\n", "9223372036854775817", "221360928884514619392"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, operation := range []struct{ path, want string }{
				{"/sum", test.sum},
				{"/multiply", test.product},
			} {
				response := postMatrix(t, operation.path, test.input)
				if response.Code != http.StatusOK || response.Body.String() != operation.want {
					t.Fatalf("%s: status %d, body %q; want %q", operation.path, response.Code, response.Body.String(), operation.want)
				}
			}
		})
	}
}

func TestSumAndMultiplyRejectInvalidMatrix(t *testing.T) {
	for _, path := range []string{"/sum", "/multiply"} {
		t.Run(path, func(t *testing.T) {
			response := postMatrix(t, path, "1,2\n3,x\n")
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid integer at row 2, column 2") {
				t.Fatalf("status %d, body %q; want invalid-integer error", response.Code, response.Body.String())
			}
		})
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
		{"empty cell", "\"\"\n", "invalid integer"},
		{"whitespace only", "\" \t\n\"\n", "invalid integer"},
		{"positive sign only", "+\n", "invalid integer"},
		{"negative sign only", "-\n", "invalid integer"},
		{"internal whitespace", "1 2\n", "invalid integer"},
		{"internal newline", "\"1\n2\"\n", "invalid integer"},
		{"decimal", "1.5\n", "invalid integer"},
		{"exponent", "1e3\n", "invalid integer"},
		{"hexadecimal", "0x10\n", "invalid integer"},
		{"digit separator", "1_000\n", "invalid integer"},
		{"non-ASCII digit", "\u0661\n", "invalid integer"},
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

func TestRequestTooLarge(t *testing.T) {
	response := postMatrix(t, "/echo", strings.Repeat("1", maxRequestBytes))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, body %q; want 413", response.Code, response.Body.String())
	}
}

func BenchmarkReadMatrix(b *testing.B) {
	row := strings.TrimSuffix(strings.Repeat("1,", 500), ",") + "\n"
	input := strings.Repeat(row, 500)
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := readMatrix(strings.NewReader(input)); err != nil {
			b.Fatal(err)
		}
	}
}
