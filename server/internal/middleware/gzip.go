package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipDecompress is middleware that decompresses request bodies
// when Content-Encoding: gzip is set. This is required by §14.2
// for the batch ingestion endpoint to accept compressed payloads.
func GzipDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, `{"error":"invalid gzip data"}`, http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = io.NopCloser(gz)
			// Remove the header so downstream handlers don't try to decompress again
			r.Header.Del("Content-Encoding")
		}
		next.ServeHTTP(w, r)
	})
}
