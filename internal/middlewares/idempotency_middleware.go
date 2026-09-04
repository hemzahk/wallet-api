package middlewares

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hemzahk/wallet-api/internal/json"
	"github.com/hemzahk/wallet-api/internal/store"
)

const (
	HeaderIdempotencyKey      = "Idempotency-Key"
	HeaderIdempotencyReplayed = "Idempotency-Replayed"
)

type IdempotencyMiddleware struct {
	idempotencyKeys store.IdempotencyKeys
}

func NewIdempotencyMiddleware(idempotencyKeys store.IdempotencyKeys) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		idempotencyKeys: idempotencyKeys,
	}
}

// responseWriter captures the response for caching
type responseWriter struct {
    http.ResponseWriter
    statusCode int
    body       bytes.Buffer
    wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
    if rw.wroteHeader {
        return
    }
    rw.wroteHeader = true
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    if !rw.wroteHeader {
        rw.WriteHeader(http.StatusOK)
    }
    rw.body.Write(b) // Capture the response body
    return rw.ResponseWriter.Write(b)
}

func (m *IdempotencyMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			json.BadRequestResponse(w, r, fmt.Errorf("idempotency key is missing"))
			return
		}

		user := r.Context().Value("user").(*store.User)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			json.InternalServerError(w,r,err)
			return
		} 

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		hash := fmt.Sprintf("%x", sha256.Sum256(bodyBytes))

		existing, err := m.idempotencyKeys.Get(r.Context(), key, user.ID)
		if err != nil && !errors.Is(err, store.ErrKeyNotFound){
			json.InternalServerError(w, r, err)
			return 
		}

		if existing != nil {
			if existing.RequestHash != hash {
				json.WriteJSONError(w, http.StatusUnprocessableEntity, "idempotency conflict")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set(HeaderIdempotencyReplayed, "true")
			w.WriteHeader(existing.StatusCode)
			w.Write(existing.ResponseBody)
			return
		}

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode: http.StatusOK,
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "idempotency-key", key)
		next.ServeHTTP(rw, r.WithContext(ctx))

		_ = m.idempotencyKeys.Create(r.Context(), store.IdempotencyKey{
				Key:          key,
                UserID:       user.ID,
                RequestHash:  hash,
                ResponseBody: rw.body.Bytes(),
                StatusCode:   rw.statusCode,
		})

	})
}