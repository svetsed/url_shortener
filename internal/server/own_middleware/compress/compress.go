package compress

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

var mayCompress = map[string]bool {
	"application/json": 	  true,
	"application/javascript": true,
	"text/css": 			  true,
	"text/html": 			  true,
	"text/plain": 			  true,
	"text/xml": 			  true,
}

type gzipWriter struct {
	http.ResponseWriter
	writer     *gzip.Writer
	statusCode int
	written    bool
	skipGzip   bool
}

func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		ResponseWriter: w,
	}
}

func (gw *gzipWriter) Write(b []byte) (int, error) {
	if !gw.written {
		gw.written = true

		// Если WriteHeader не был вызван явно, устанавливаем 200
		if gw.statusCode == 0 {
			gw.statusCode = http.StatusOK
		}

		// Определяем Content-Type если не установлен
		contentType := gw.Header().Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(b)
			gw.Header().Set("Content-Type", contentType)
		}

		// Проверяем нужно ли сжимать
		if shouldCompress(contentType) && gw.statusCode < 300 && len(b) > 8 && !gw.skipGzip {
			gw.Header().Set("Content-Encoding", "gzip")
			gw.Header().Del("Content-Length")
			
			// Создаем gzip.Writer только когда точно решили сжимать
			gz, err := gzip.NewWriterLevel(gw.ResponseWriter, gzip.BestSpeed)
			if err != nil {
				// Если не удалось создать writer, пишем без сжатия
				gw.skipGzip = true
				gw.ResponseWriter.WriteHeader(gw.statusCode)
				return gw.ResponseWriter.Write(b)
			}
			gw.writer = gz
			gw.ResponseWriter.WriteHeader(gw.statusCode)
			return gw.writer.Write(b)
		}

		// Не сжимаем
		gw.skipGzip = true
		gw.ResponseWriter.WriteHeader(gw.statusCode)
		return gw.ResponseWriter.Write(b)
	}

	// После того как решение принято, продолжаем запись
	if gw.writer != nil {
		return gw.writer.Write(b)
	}
	return gw.ResponseWriter.Write(b)
}

func (gw *gzipWriter) WriteHeader(statusCode int) {
	if gw.written {
		return
	}
	// Сохраняем статус, но не пишем заголовки пока не знаем Content-Type
	gw.statusCode = statusCode

	// Для статусов без тела — отправляем сразу
    if statusCode == http.StatusNoContent || 
       statusCode == http.StatusNotModified ||
       statusCode < 200 || statusCode >= 300 {
        gw.written = true
        gw.skipGzip = true
        gw.ResponseWriter.WriteHeader(statusCode)
    }
}

func (gw *gzipWriter) Close() error {
	if gw.writer != nil {
		return gw.writer.Close()
	}
	return nil
}

func (gw *gzipWriter) Header() http.Header {
	return gw.ResponseWriter.Header()
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentEncoding := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
		if contentEncoding == "gzip" {
			if r.Body != nil && r.Body != http.NoBody {
				cr, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, "invalid gzip data", http.StatusBadRequest)
					return
				}
				r.Body = cr
				defer cr.Close()
				r.Header.Del("Content-Encoding")
			}
		}

		if !clientSupportsGzip(r.Header.Values("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		gzw := newGzipWriter(w)
		defer gzw.Close()

		next.ServeHTTP(gzw, r)
	})
}

func shouldCompress(contentType string) bool {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	contentType = strings.TrimSpace(strings.ToLower(contentType))

	_, ok := mayCompress[contentType]

	return ok
}

// example
// Accept-Encoding: gzip;q=0.8, deflate;q=0.6, br;q=0.9
// Accept-Encoding: gzip;q=1.0, identity; q=0.5, *;q=0
// q=0 => No | q>0 => Yes
// now without preference weight
func parseAcceptEncoding(parts []string) map[string]float64 {
	res := make(map[string]float64)

	for _, part := range parts {
		encodingAndParams := strings.Split(strings.TrimSpace(part), ";")
		encoding := strings.ToLower((strings.TrimSpace(encodingAndParams[0])))

		weight := 1.0

		for i:= 1; i < len(encodingAndParams); i++ {
			param := strings.TrimSpace(encodingAndParams[i])
			if strings.HasPrefix(param, "q=") {
				qValue := strings.TrimPrefix(param, "q=")
				if f, err := strconv.ParseFloat(qValue, 64); err == nil {
					weight = f
				}
			}
		}

		res[encoding] = weight
	}

	return res
}

func clientSupportsGzip(acceptEncoding []string) bool {
	encodings := parseAcceptEncoding(acceptEncoding)

	// Проверяем явный запрет gzip (q=0)
	if weight, ok := encodings["gzip"]; ok && weight == 0 {
		return false
	}

    // Если gzip есть с положительным весом — проверяем, не предпочитает ли клиент identity
    if gzipWeight, ok := encodings["gzip"]; ok && gzipWeight > 0 {
        // Если identity указан и имеет больший вес — не сжимаем
        if identityWeight, hasIdentity := encodings["identity"]; hasIdentity {
            if identityWeight > gzipWeight {
                return false
            }
        }
        return true
    }

	// Проверяем wildcard
    if starWeight, ok := encodings["*"]; ok && starWeight > 0 {
        if identityWeight, hasIdentity := encodings["identity"]; hasIdentity && identityWeight >= starWeight {
            return false
        }
        return true
    }

    return false
}