package ownmiddleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type logResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	if err != nil {
		return 0, err
	}

	w.responseData.size += size
	return size, err
}

func (w *logResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.status = statusCode
}

func LoggingMiddleware(sugarLog *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := middleware.GetReqID(r.Context())

			respData := &responseData{}

			lw := logResponseWriter{
				ResponseWriter: w,
				responseData: respData,
			}

			next.ServeHTTP(&lw, r)

			duration := time.Since(start)

			sugarLog.Infoln(
				"reqID", reqID,
				"uri", r.RequestURI,
				"method", r.Method,
				"status", respData.status,
				"duration", duration,
				"size", respData.size,
			)
		})
	}
}