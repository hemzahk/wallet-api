package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hemzahk/wallet-api/internal/json"
	"github.com/hemzahk/wallet-api/internal/ratelimiter"
)

type rateLimiterMiddleware struct {
	limiter ratelimiter.Limiter
}

func NewRateLimiterMiddleware(limiter ratelimiter.Limiter) *rateLimiterMiddleware {
	return &rateLimiterMiddleware{
		limiter: limiter,
	}
}

func (m *rateLimiterMiddleware) RateLimiterMiddleware(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allow, retryAfter := m.limiter.Allow(middleware.GetClientIPAddr(r.Context()).String()); !allow {
			json.RateLimitExceededResponse(w, r, retryAfter.String())
			return 
		}

		next.ServeHTTP(w,r)
	})
}