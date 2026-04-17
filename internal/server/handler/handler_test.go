package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/svetsed/url_shortener/internal/config"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/service"
	"github.com/svetsed/url_shortener/internal/storage/mock"
	"go.uber.org/zap"
)

var sugarLog = zap.NewNop().Sugar()

func TestCreateShortURLHandler(t *testing.T) {
	cfg := config.NewDefaultConfig()

	type want struct {
		code        int
		response    string
		contentType string
	}

	tests := []struct {
		name   string
		want   want
		method string
		body   []byte
		setup  func(*mock.MockStorage)
	}{
		{
			name: "valid POST request with new URL",
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/", // check only prefix
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("https://example.com"),
		},
		{
			name: "method not allowed - GET request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "method not allowed",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodGet,
			body:   []byte("https://example.com"),
		},
		{
			name: "method not allowed - PUT request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "method not allowed",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPut,
			body:   []byte("https://example.com"),
		},
		{
			name: "method not allowed - DELETE request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "method not allowed",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodDelete,
			body:   []byte("https://example.com"),
		},
		{
			name: "invalid URL - empty body",
			want: want{
				code:        http.StatusBadRequest,
				response:    "bad request",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte(""),
		},
		{
			name: "invalid URL",
			want: want{
				code:        http.StatusBadRequest,
				response:    "bad request",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("not-a-valid url"),
		},
		{
			name: "invalid URL - without protocol",
			want: want{
				code:        http.StatusBadRequest,
				response:    "bad request",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("example.com"),
		},
		{
			name: "existing URL",
			want: want{
				code:        http.StatusConflict,
				response:    "http://localhost:8080/existing123",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("https://existing-example.com"),
			setup: func(ms *mock.MockStorage) {
				existingURL := &model.URL{
					ShortURL:    "existing123",
					OriginalURL: "https://existing-example.com",
				}
				ms.Save(existingURL)
			},
		},
		{
			name: "very long original URL",
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("https://example.com/" + strings.Repeat("a", 1000)),
		},
		{
			name: "URL with param and fragment",
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/",
				contentType: "text/plain; charset=utf-8",
			},
			method: http.MethodPost,
			body:   []byte("https://example.com/path?param=value&other=123#fragment"),
		},

		// TODO "duplicate short URL generation - should retry
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSt := mock.NewMockStorage()
			if test.setup != nil {
				test.setup(mockSt)
			}
			serv := service.NewService(mockSt)
			h := NewHandler(serv, cfg, sugarLog)

			server := httptest.NewServer(http.HandlerFunc(h.CreateShortURLHandler))
			defer server.Close()

			client := resty.New()

			var resp *resty.Response
			var err error
			switch test.method {
			case http.MethodPost:
				resp, err = client.R().
					SetHeader("Content-Type", "text/plain").
					SetBody(test.body).
					Post(server.URL + "/")
			case http.MethodGet:
				resp, err = client.R().
					Get(server.URL + "/")
			case http.MethodPut:
				resp, err = client.R().
					SetHeader("Content-Type", "text/plain").
					SetBody(test.body).
					Put(server.URL + "/")
			case http.MethodDelete:
				resp, err = client.R().
					Delete(server.URL + "/")
			}

			assert.NoError(t, err, "Request failed")
			assert.Equal(t, test.want.code, resp.StatusCode(), "Status code mismatch")
			assert.Equal(t, test.want.contentType, resp.Header().Get("Content-Type"), "Content-Type mismatch")

			respBody := resp.String()
			assert.NoError(t, err, "Failed to read response body")

			if test.want.response != respBody {
				if test.want.response == "http://localhost:8080/" {
					assert.True(t, strings.HasPrefix(respBody, "http://localhost:8080/"), "Response should start with base URL")
					assert.True(t, len(respBody) > len("http://localhost:8080/"), "Response should contain short URL path")
				} else {
					t.Errorf("unexpected response: want - %s, but received - %s", test.want.response, respBody)
				}
			}
		})
	}
}

func TestRedirectToOrigURLHandler(t *testing.T) {
	cfg := config.NewDefaultConfig()

	type want struct {
		code        int
		location    string
		contentType string
	}

	tests := []struct {
		name      string
		want      want
		method    string
		pathValue string
		setup     func(*mock.MockStorage)
		checkBody bool // if error have
	}{
		{
			name: "base case - successful redirect",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "https://example.com",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "abc123",
			setup: func(ms *mock.MockStorage) {
				url := &model.URL{
					ShortURL:    "abc123",
					OriginalURL: "https://example.com",
				}
				ms.Save(url)
			},
			checkBody: false,
		},
		{
			name: "method not allowed - POST request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodPost,
			pathValue: "abc123",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "method not allowed - PUT request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodPut,
			pathValue: "abc123",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "method not allowed - DELETE request",
			want: want{
				code:        http.StatusMethodNotAllowed,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodDelete,
			pathValue: "abc123",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "empty ID in path",
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "non-existent short URL",
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "nonexistent",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "very long pathValue",
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: strings.Repeat("a", 1000),
			setup:     nil,
			checkBody: true,
		},
		{
			name: "ID with special characters",
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "abc@#$123",
			setup:     nil,
			checkBody: true,
		},
		{
			name: "redirect to URL with params and fragment",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "https://example.com/path?param=value&other=123#fragment",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "special123",
			setup: func(ms *mock.MockStorage) {
				url := &model.URL{
					ShortURL:    "special123",
					OriginalURL: "https://example.com/path?param=value&other=123#fragment",
				}
				ms.Save(url)
			},
			checkBody: false,
		},
		{
			name: "redirect to HTTP (not HTTPS) URL",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "http://example.com",
				contentType: "text/plain; charset=utf-8",
			},
			method:    http.MethodGet,
			pathValue: "http123",
			setup: func(ms *mock.MockStorage) {
				url := &model.URL{
					ShortURL:    "http123",
					OriginalURL: "http://example.com",
				}
				ms.Save(url)
			},
			checkBody: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSt := mock.NewMockStorage()
			if test.setup != nil {
				test.setup(mockSt)
			}

			serv := service.NewService(mockSt)
			h := NewHandler(serv, cfg, sugarLog)

			r := httptest.NewRequest(test.method, "/{id}", nil)
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("id", test.pathValue)

			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))

			w := httptest.NewRecorder()

			h.RedirectToOrigURLHandler(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, test.want.code, res.StatusCode, "Status code mismatch")
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"), "Content-Type mismatch")

			if test.want.code == http.StatusTemporaryRedirect {
				location := res.Header.Get("Location")
				assert.Equal(t, test.want.location, location, "Location header mismatch")
			}

			if test.checkBody {
				body, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				var expMsg string
				switch test.want.code {
				case http.StatusMethodNotAllowed:
					expMsg = "method not allowed\n"
				case http.StatusBadRequest:
					expMsg = "bad request\n"
				}

				assert.Equal(t, expMsg, string(body), "Error message mismatch")
			}
		})
	}
}

func TestCreateShortURLHandlerFromJSON(t *testing.T) {
	cfg := config.NewDefaultConfig()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "base case - POST request with valid URL in JSON",
			method:         http.MethodPost,
			body:           `{"url": "https://example.com"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           `{"url": "https://example.com"}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "empty body",
			method:         http.MethodPost,
			body:           ``,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `{invalid json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no field url",
			method:         http.MethodPost,
			body:           `{"link": "https://example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty url",
			method:         http.MethodPost,
			body:           `{"url": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockSt := mock.NewMockStorage()
			serv := service.NewService(mockSt)
			h := NewHandler(serv, cfg, sugarLog)

			r := httptest.NewRequest(test.method, "/api/shorten", bytes.NewReader([]byte(test.body)))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateShortURLHandlerFromJSON(w, r)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, test.expectedStatus, res.StatusCode, "Status code mismatch")
			// assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"), "Content-Type mismatch")

			_, err := io.ReadAll(res.Body)
			assert.NoError(t, err, "Failed to read response body")

			// if test.want.response != string(respBody) {
			// 	if test.want.response == "http://localhost:8080/" {
			// 		assert.True(t, strings.HasPrefix(string(respBody), "http://localhost:8080/"), "Response should start with base URL")
			// 		assert.True(t, len(string(respBody)) > len("http://localhost:8080/"), "Response should contain short URL path")
			// 	} else {
			// 		t.Errorf("unexpected response: want - %s, but received - %s", test.want.response, string(respBody))
			// 	}
			// }
		})
	}
}
