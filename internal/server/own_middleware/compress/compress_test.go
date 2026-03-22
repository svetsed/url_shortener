package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		acceptEncoding     []string
		contentEncoding    string
		responseBody       string
		responseContentType string
		expectCompressed   bool
		expectStatusCode   int
	}{
		{
			name:               "client supports gzip, compressible content",
			acceptEncoding:     []string{"gzip"},
			responseBody:       `{"message": "hello world"}`,
			responseContentType: "application/json",
			expectCompressed:   true,
			expectStatusCode:   http.StatusOK,
		},
		{
			name:               "client supports gzip, non-compressible content",
			acceptEncoding:     []string{"gzip"},
			responseBody:       "image data",
			responseContentType: "image/jpeg",
			expectCompressed:   false,
			expectStatusCode:   http.StatusOK,
		},
		{
			name:               "client does not support gzip",
			acceptEncoding:     []string{"deflate"},
			responseBody:       `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:   false,
			expectStatusCode:   http.StatusOK,
		},
		{
			name:               "client supports gzip with weight 0",
			acceptEncoding:     []string{"gzip;q=0"},
			responseBody:       `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:   false,
			expectStatusCode:   http.StatusOK,
		},
		{
			name:               "client supports wildcard",
			acceptEncoding:     []string{"*"},
			responseBody:       `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:   true,
			expectStatusCode:   http.StatusOK,
		},
		{
			name:               "client sends compressed request",
			contentEncoding:    "gzip",
			acceptEncoding:     []string{"gzip"},
			responseBody:       `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:   true,
			expectStatusCode:   http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.responseContentType)
				w.WriteHeader(tt.expectStatusCode)
				w.Write([]byte(tt.responseBody))
			})
			
			// Создаем middleware
			middleware := GzipMiddleware(handler)
			
			// Создаем запрос
			var body io.Reader
			if tt.contentEncoding == "gzip" {
				var buf bytes.Buffer
				gzWriter := gzip.NewWriter(&buf)
				gzWriter.Write([]byte(tt.responseBody))
				gzWriter.Close()
				body = &buf
			} else {
				body = strings.NewReader(tt.responseBody)
			}
			
			req := httptest.NewRequest("POST", "/", body)
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			if len(tt.acceptEncoding) > 0 {
				for _, ae := range tt.acceptEncoding {
					req.Header.Add("Accept-Encoding", ae)
				}
			}
			
			// Создаем recorder
			rw := httptest.NewRecorder()
			
			// Выполняем запрос
			middleware.ServeHTTP(rw, req)
			
			// Проверяем статус код
			if rw.Code != tt.expectStatusCode {
				t.Errorf("status code = %d, want %d", rw.Code, tt.expectStatusCode)
			}
			
			// Проверяем сжатие
			contentEncoding := rw.Header().Get("Content-Encoding")
			if tt.expectCompressed {
				if contentEncoding != "gzip" {
					t.Errorf("expected gzip encoding, got %q", contentEncoding)
				}
				
				// Проверяем, что тело действительно сжато
				gzReader, err := gzip.NewReader(rw.Body)
				if err != nil {
					t.Errorf("failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()
				
				decompressed, err := io.ReadAll(gzReader)
				if err != nil {
					t.Errorf("failed to decompress: %v", err)
				}
				
				if string(decompressed) != tt.responseBody {
					t.Errorf("decompressed body = %q, want %q", string(decompressed), tt.responseBody)
				}
			} else {
				if contentEncoding == "gzip" {
					t.Errorf("unexpected gzip encoding")
				}
				
				// Проверяем, что тело не сжато
				if rw.Body.String() != tt.responseBody {
					t.Errorf("body = %q, want %q", rw.Body.String(), tt.responseBody)
				}
			}
		})
	}
}

func TestShouldCompress(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"JSON with charset", "application/json; charset=utf-8", true},
		{"Plain JSON", "application/json", true},
		{"HTML", "text/html", true},
		{"CSS", "text/css", true},
		{"Plain text", "text/plain", true},
		{"XML", "text/xml", true},
		{"JavaScript", "application/javascript", true},
		{"Image", "image/jpeg", false},
		{"Video", "video/mp4", false},
		{"Empty", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCompress(tt.contentType)
			if result != tt.expected {
				t.Errorf("shouldCompress(%q) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}