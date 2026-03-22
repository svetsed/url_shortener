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
	writer 			   *gzip.Writer
	statusCode 		   int
	contentTypeChecked bool
}

func (gw *gzipWriter) Write(b []byte) (int, error) {
	if !gw.contentTypeChecked {
		// this means the WriteHeader was not called
		contentType := http.DetectContentType(b)
		gw.Header().Set("Content-Type", contentType)

		if shouldCompress(contentType) {
			gz, err := gzip.NewWriterLevel(gw.ResponseWriter, gzip.BestSpeed)
			if err != nil {
				return 0, err
			}

			gw.writer = gz
			gw.Header().Set("Content-Encoding", "gzip")
		} else {
			gw.writer = nil
			gw.Header().Del("Content-Encoding")
		}

		if gw.statusCode == 0 {
			gw.statusCode = http.StatusOK
		}

		gw.contentTypeChecked = true
		gw.ResponseWriter.WriteHeader(gw.statusCode)
	}

	if gw.writer != nil {
		return gw.writer.Write(b)
	}

	return gw.ResponseWriter.Write(b)
}

func (gw *gzipWriter) WriteHeader(statusCode int) {
	gw.statusCode = statusCode

	if !gw.contentTypeChecked {
		contentType := gw.Header().Get("Content-Type")

		if shouldCompress(contentType) {
			// Создаем gzip writer только если нужно сжимать
			gz, err := gzip.NewWriterLevel(gw.ResponseWriter, gzip.BestSpeed)
			if err != nil {
				return
			}

			gw.writer = gz
			gw.Header().Set("Content-Encoding", "gzip")
			
		} else {
			gw.writer = nil
			gw.Header().Del("Content-Encoding")
		}

		gw.contentTypeChecked = true
	}

	gw.ResponseWriter.WriteHeader(statusCode)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// checks, that client send to server compress data in the gzip format
		encodings := strings.Split(r.Header.Get("Content-Encoding"), ",")
		for _, enc := range encodings {
			enc = strings.TrimSpace(strings.ToLower(enc))
			if enc == "gzip" {
				cr, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, "server error", http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
				break
			}
		}

		// checks, that client supports gzip
		if !clientSupportsGzip(r.Header.Values("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		gzw := gzipWriter{
			ResponseWriter: w,
			writer: nil,
		}

		next.ServeHTTP(&gzw, r)

		if gzw.writer != nil {
			gzw.writer.Close()
		}
	})
}

func shouldCompress(contentType string) bool {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

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

	if weight, ok := encodings["gzip"]; ok && weight > 0 {
		return true
	}

	if weight, ok := encodings["*"]; ok && weight > 0 {
		return true
	}

	return false
}