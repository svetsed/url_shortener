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
		name                string
		acceptEncoding      []string
		contentEncoding     string
		requestBody         string
		responseBody        string
		responseContentType string
		responseStatusCode  int
		expectCompressed    bool
		expectStatusCode    int
		expectError         bool
		invalidGzip 		bool
	}{
		{
			name:                "client supports gzip, compressible content",
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"message": "hello world"}`,
			responseContentType: "application/json",
			expectCompressed:    true,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "client supports gzip, non-compressible content",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "image data",
			responseContentType: "image/jpeg",
			expectCompressed:    false,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "client does not support gzip",
			acceptEncoding:      []string{"deflate"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:    false,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "client supports gzip with weight 0",
			acceptEncoding:      []string{"gzip;q=0"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:    false,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "client supports wildcard",
			acceptEncoding:      []string{"*"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:    true,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "client sends compressed request",
			contentEncoding:     "gzip",
			requestBody:         `{"message": "hello"}`,
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:    true,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "empty compressed request body",
			contentEncoding:     "gzip",
			requestBody:         "",
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"message": "ok"}`,
			responseContentType: "application/json",
			expectCompressed:    false,
			expectStatusCode:    http.StatusBadRequest,
			expectError:         true,
		},
		{
			name:                "invalid gzip data in request",
			contentEncoding:     "gzip",
			invalidGzip: 		 true,
			requestBody:         "invalid gzip data",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "",
			responseContentType: "",
			expectCompressed:    false,
			expectStatusCode:    http.StatusBadRequest,
			expectError:         true,
		},
		{
			name:                "response with error status code",
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"error": "not found"}`,
			responseContentType: "application/json",
			responseStatusCode:  http.StatusNotFound,
			// сжимает если status < 300, 404 >= 300, значит НЕ сжимает
			expectCompressed:    false,
			expectStatusCode:    http.StatusNotFound,
		},
		{
			name:                "multiple accept-encoding headers",
			acceptEncoding:      []string{"deflate;q=0.5", "gzip;q=0.8", "br;q=0.3"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			expectCompressed:    true,
			expectStatusCode:    http.StatusOK,
		},
		{
			name:                "identity with higher weight than gzip",
			acceptEncoding:      []string{"gzip;q=0.5", "identity;q=1.0"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			// identity с весом 1.0 > gzip с весом 0.5, клиент предпочитает без сжатия
			expectCompressed:    false,
			expectStatusCode:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Проверяем, что тело запроса корректно обработано
				if tt.contentEncoding == "gzip" && !tt.expectError {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("failed to read body: %v", err)
					}
					if string(body) != tt.requestBody {
						t.Errorf("request body = %q, want %q", string(body), tt.requestBody)
					}
				}

				// Устанавливаем статус код (по умолчанию 200)
				statusCode := tt.responseStatusCode
				if statusCode == 0 {
					statusCode = http.StatusOK
				}

				if tt.responseContentType != "" {
					w.Header().Set("Content-Type", tt.responseContentType)
				}
				w.WriteHeader(statusCode)
				if tt.responseBody != "" {
					w.Write([]byte(tt.responseBody))
				}
			})

			middleware := GzipMiddleware(handler)

			// Подготовка тела запроса
			var body io.Reader
			if tt.contentEncoding == "gzip" {
				if tt.invalidGzip {
        			body = strings.NewReader(tt.requestBody)
				} else {
					var buf bytes.Buffer
					if tt.requestBody != "" {
						gzWriter := gzip.NewWriter(&buf)
						_, err := gzWriter.Write([]byte(tt.requestBody))
						if err != nil {
							t.Fatalf("failed to write gzip data: %v", err)
						}
						gzWriter.Close()
						body = &buf
					} else {
						// Пустое тело — невалидный gzip
						body = strings.NewReader("")
					}
				}
			} else {
				if tt.requestBody != "" {
					body = strings.NewReader(tt.requestBody)
				} else {
					body = strings.NewReader(tt.responseBody)
				}
			}

			req := httptest.NewRequest("POST", "/", body)
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			for _, ae := range tt.acceptEncoding {
				req.Header.Add("Accept-Encoding", ae)
			}

			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)

			// Проверяем статус код
			if rw.Code != tt.expectStatusCode {
				t.Errorf("status code = %d, want %d", rw.Code, tt.expectStatusCode)
			}

			// Если ожидаем ошибку, дальше не проверяем
			if tt.expectError {
				return
			}

			// Проверяем сжатие ответа
			contentEncoding := rw.Header().Get("Content-Encoding")
			if tt.expectCompressed {
				if contentEncoding != "gzip" {
					t.Errorf("expected gzip encoding, got %q", contentEncoding)
				}

				// Проверяем, что тело действительно сжато
				if rw.Body.Len() > 0 {
					gzReader, err := gzip.NewReader(rw.Body)
					if err != nil {
						t.Fatalf("failed to create gzip reader: %v", err)
					}
					defer gzReader.Close()

					decompressed, err := io.ReadAll(gzReader)
					if err != nil {
						t.Fatalf("failed to decompress: %v", err)
					}

					if string(decompressed) != tt.responseBody {
						t.Errorf("decompressed body = %q, want %q", string(decompressed), tt.responseBody)
					}
				} else if tt.responseBody != "" {
					t.Errorf("expected non-empty compressed body, got empty")
				}
			} else {
				if contentEncoding == "gzip" {
					t.Errorf("unexpected gzip encoding")
				}

				if rw.Body.String() != tt.responseBody {
					t.Errorf("body = %q, want %q", rw.Body.String(), tt.responseBody)
				}
			}

			// Дополнительная проверка: убеждаемся, что Content-Length удален при сжатии
			if tt.expectCompressed {
				if rw.Header().Get("Content-Length") != "" {
					t.Errorf("Content-Length should be removed when compressing, but got %q",
						rw.Header().Get("Content-Length"))
				}
			}
		})
	}
}

// TestEdgeCases тестирует граничные случаи
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name                string
		acceptEncoding      []string
		responseBody        string
		responseContentType string
		setContentType      bool
		statusCode          int
		expectCompressed    bool
	}{
		{
			name:                "empty response body",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "",
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusOK,
			// Пустое тело не сжимаем (нет смысла)
			expectCompressed:    false,
		},
		{
			name:                "very small response (less than compression threshold)",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "a", // 1 байт
			responseContentType: "text/plain",
			setContentType:      true,
			statusCode:          http.StatusOK,
			// Маленькие ответы не сжимаем (overhead gzip > выигрыша)
			expectCompressed:    false,
		},
		{
			name:                "status code 204 No Content",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "",
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusNoContent,
			expectCompressed:    false,
		},
		{
			name:                "status code 304 Not Modified",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "",
			responseContentType: "text/html",
			setContentType:      true,
			statusCode:          http.StatusNotModified,
			expectCompressed:    false,
		},
		{
			name:                "status code 500 Internal Server Error",
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"error": "internal server error"}`,
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusInternalServerError,
			// 500 >= 300, значит НЕ сжимаем
			expectCompressed:    false,
		},
		{
			name:                "content type not set before write",
			acceptEncoding:      []string{"gzip"},
			responseBody:        `{"message": "test"}`,
			responseContentType: "", // не устанавливаем явно
			setContentType:      false,
			statusCode:          http.StatusOK,
			// DetectContentType определит как application/json
			expectCompressed:    true,
		},
		{
			name:                "content type with charset",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "<html><body>test</body></html>",
			responseContentType: "text/html; charset=utf-8",
			setContentType:      true,
			statusCode:          http.StatusOK,
			expectCompressed:    true,
		},
		{
			name:                "content type with parameters",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "test content",
			responseContentType: "text/plain; charset=utf-8; foo=bar",
			setContentType:      true,
			statusCode:          http.StatusOK,
			expectCompressed:    true,
		},
		{
			name:                "multiple accept-encoding headers",
			acceptEncoding:      []string{"deflate", "gzip", "br"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusOK,
			expectCompressed:    true,
		},
		{
			name:                "accept-encoding with weights",
			acceptEncoding:      []string{"deflate;q=0.5", "gzip;q=0.8", "br;q=0.9"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusOK,
			expectCompressed:    true,
		},
		{
			name:                "invalid accept-encoding weight",
			acceptEncoding:      []string{"gzip;q=invalid"},
			responseBody:        `{"message": "hello"}`,
			responseContentType: "application/json",
			setContentType:      true,
			statusCode:          http.StatusOK,
			// должен использовать значение по умолчанию 1.0
			expectCompressed:    true,
		},
		{
			name:                "no content-type header",
			acceptEncoding:      []string{"gzip"},
			responseBody:        "plain text without content type",
			responseContentType: "",
			setContentType:      false,
			statusCode:          http.StatusOK,
			// определится как text/plain через DetectContentType
			expectCompressed:    true,
		},
		{
			name:                "binary content detected as non-compressible",
			acceptEncoding:      []string{"gzip"},
			responseBody:        string([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}), // PNG signature
			responseContentType: "",
			setContentType:      false,
			statusCode:          http.StatusOK,
			// должен определиться как image/png (или application/octet-stream)
			expectCompressed:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.setContentType && tt.responseContentType != "" {
					w.Header().Set("Content-Type", tt.responseContentType)
				}
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					w.Write([]byte(tt.responseBody))
				}
			})

			middleware := GzipMiddleware(handler)
			req := httptest.NewRequest("GET", "/", nil)

			for _, ae := range tt.acceptEncoding {
				req.Header.Add("Accept-Encoding", ae)
			}

			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)

			// Проверяем статус код
			if rw.Code != tt.statusCode {
				t.Errorf("status code = %d, want %d", rw.Code, tt.statusCode)
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
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()

				decompressed, err := io.ReadAll(gzReader)
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}

				if string(decompressed) != tt.responseBody {
					t.Errorf("decompressed body = %q, want %q", string(decompressed), tt.responseBody)
				}
			} else {
				if contentEncoding == "gzip" {
					t.Errorf("unexpected gzip encoding")
				}

				if rw.Body.String() != tt.responseBody {
					t.Errorf("body = %q, want %q", rw.Body.String(), tt.responseBody)
				}
			}
		})
	}
}

// TestShouldCompressEdgeCases тестирует граничные случаи shouldCompress
func TestShouldCompressEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"empty string", "", false},
		{"only charset", "; charset=utf-8", false},
		{"with multiple parameters", "application/json; charset=utf-8; version=1", true},
		{"with spaces", "  application/json  ;  charset=utf-8  ", true},
		{"unknown type", "application/unknown", false},
		{"case insensitive", "APPLICATION/JSON", true},
		{"text with parameters", "text/html; charset=utf-8; q=0.9", true},
		{"wildcard", "*/*", false},
		{"application wildcard", "application/*", false},
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

// TestLargeData тестирует обработку больших данных
func TestLargeData(t *testing.T) {
	tests := []struct {
		name             string
		dataSize         int // в байтах
		contentType      string
		acceptEncoding   []string
		expectCompressed bool
	}{
		{
			name:             "1KB data",
			dataSize:         1024,
			contentType:      "application/json",
			acceptEncoding:   []string{"gzip"},
			expectCompressed: true,
		},
		{
			name:             "10KB data",
			dataSize:         10 * 1024,
			contentType:      "application/json",
			acceptEncoding:   []string{"gzip"},
			expectCompressed: true,
		},
		{
			name:             "100KB data",
			dataSize:         100 * 1024,
			contentType:      "application/json",
			acceptEncoding:   []string{"gzip"},
			expectCompressed: true,
		},
		{
			name:             "1MB data",
			dataSize:         1024 * 1024,
			contentType:      "application/json",
			acceptEncoding:   []string{"gzip"},
			expectCompressed: true,
		},
		{
			name:             "large non-compressible data",
			dataSize:         1024 * 1024,
			contentType:      "image/jpeg",
			acceptEncoding:   []string{"gzip"},
			expectCompressed: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Генерируем данные нужного размера
			var data string
			if tt.contentType == "application/json" {
				// Используем повторяющийся JSON для лучшего сжатия
				pattern := `{"id":1234567890,"name":"test","data":"` + 
					strings.Repeat("x", 100) + `"},`
				repetitions := tt.dataSize / len(pattern)
				if repetitions < 1 {
					repetitions = 1
				}
				data = "[" + strings.Repeat(pattern, repetitions)
				data = data[:len(data)-1] + "]"
				if len(data) > tt.dataSize {
					data = data[:tt.dataSize]
				}
			} else {
				// Несжимаемые данные (случайные байты)
				b := make([]byte, tt.dataSize)
				for i := range b {
					b[i] = byte(i % 256)
				}
				data = string(b)
			}
			
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Write([]byte(data))
			})
			
			middleware := GzipMiddleware(handler)
			req := httptest.NewRequest("GET", "/", nil)
			for _, ae := range tt.acceptEncoding {
				req.Header.Add("Accept-Encoding", ae)
			}
			
			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)
			
			contentEncoding := rw.Header().Get("Content-Encoding")
			
			if tt.expectCompressed {
				if contentEncoding != "gzip" {
					t.Errorf("expected gzip encoding, got %q", contentEncoding)
				}
				
				// Проверяем, что сжатие действительно уменьшило размер (для сжимаемых данных)
				if tt.contentType == "application/json" && len(data) > 1024 {
					compressedSize := rw.Body.Len()
					if compressedSize >= len(data) {
						t.Logf("Warning: compressed size (%d) is not smaller than original (%d)", 
							compressedSize, len(data))
					}
				}
				
				// Распаковываем и проверяем содержимое
				gzReader, err := gzip.NewReader(rw.Body)
				if err != nil {
					t.Fatalf("failed to create gzip reader: %v", err)
				}
				defer gzReader.Close()
				
				decompressed, err := io.ReadAll(gzReader)
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}
				
				if len(decompressed) != len(data) {
					t.Errorf("decompressed size = %d, want %d", len(decompressed), len(data))
				}
			} else {
				if contentEncoding == "gzip" {
					t.Errorf("unexpected gzip encoding for non-compressible data")
				}
			}
			
			// Проверяем, что нет паники и запрос завершился успешно
			if rw.Code != http.StatusOK {
				t.Errorf("status code = %d, want %d", rw.Code, http.StatusOK)
			}
		})
	}
}

// TestMultipleWriteCalls тестирует множественные вызовы Write
func TestMultipleWriteCalls(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "part1"`))
		w.Write([]byte(` "part2"}`))
	})
	
	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	
	rw := httptest.NewRecorder()
	middleware.ServeHTTP(rw, req)
	
	// Проверяем, что ответ сжат
	if rw.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip encoding")
	}
	
	// Распаковываем и проверяем содержимое
	gzReader, err := gzip.NewReader(rw.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()
	
	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	
	expected := `{"message": "part1" "part2"}`
	if string(decompressed) != expected {
		t.Errorf("decompressed body = %q, want %q", string(decompressed), expected)
	}
}

// TestWriteHeaderAfterWrite тестирует вызов WriteHeader после Write
func TestWriteHeaderAfterWrite(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "test"}`))
		w.WriteHeader(http.StatusCreated) // Это не должно изменить статус
	})
	
	middleware := GzipMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	
	rw := httptest.NewRecorder()
	middleware.ServeHTTP(rw, req)
	
	// Должен быть StatusOK, так как Write был вызван до WriteHeader
	if rw.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rw.Code, http.StatusOK)
	}
}

// TestConcurrentRequests тестирует конкурентные запросы
func TestConcurrentRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "hello"}`))
	})
	
	middleware := GzipMiddleware(handler)
	
	// Запускаем 100 конкурентных запросов
	concurrency := 100
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in concurrent request: %v", r)
				}
				done <- true
			}()
			
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			
			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)
			
			// Проверяем результат
			if rw.Code != http.StatusOK {
				t.Errorf("status code = %d, want %d", rw.Code, http.StatusOK)
			}
			
			if rw.Header().Get("Content-Encoding") != "gzip" {
				t.Error("expected gzip encoding")
			}
		}()
	}
	
	// Ждем завершения всех запросов
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// BenchmarkGzipMiddleware бенчмарк для оценки производительности
func BenchmarkGzipMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "hello world", "data": "` + 
			strings.Repeat("x", 1000) + `"}`))
	})
	
	middleware := GzipMiddleware(handler)
	
	b.Run("with compression", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)
		}
	})
	
	b.Run("without compression", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			rw := httptest.NewRecorder()
			middleware.ServeHTTP(rw, req)
		}
	})
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