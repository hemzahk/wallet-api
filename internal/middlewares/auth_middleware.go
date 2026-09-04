package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/auth"
	"github.com/hemzahk/wallet-api/internal/json"
	"github.com/hemzahk/wallet-api/internal/store"
)

type AuthMiddleware struct {
	authenticator auth.Authenticator
	users store.Users
	roles store.Roles
}

func NewAuthMiddleware(auth auth.Authenticator, users store.Users, roles store.Roles) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: auth,
		users: users,
		roles: roles,
	}
}

func (m *AuthMiddleware) AuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			json.UnauthorizedErrorResponse(w,r, fmt.Errorf("authorization header is missing"))
			return 
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			json.UnauthorizedErrorResponse(w,r, fmt.Errorf("authorization header is malformed"))
			return 
		}

		token := parts[1]
		jwtToken, err := m.authenticator.ValidateToken(token)
		if err != nil {
			json.UnauthorizedErrorResponse(w,r,err)
			return 
		}

		claims, _ := jwtToken.Claims.(jwt.MapClaims)
		subStr, err := claims.GetSubject()
		if err != nil || subStr == "" {
			json.UnauthorizedErrorResponse(w,r,err)
			return 
		}

		userID, err := uuid.Parse(subStr)
		if err != nil {
			json.UnauthorizedErrorResponse(w,r,err)
			return 
		}

		ctx := r.Context()
		
		user, err := m.users.GetByID(ctx, userID)
		if err != nil {
			json.UnauthorizedErrorResponse(w,r,err)
			return 
		}

		ctx = context.WithValue(ctx, "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) CheckRequiredRole(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, err := m.roles.GetByName(r.Context(), requiredRole)
		if err != nil {
			json.ForbiddenResponse(w, r)
			return
		}

		user := r.Context().Value("user").(*store.User)

		if role.Name != user.Role.Name {
			json.ForbiddenResponse(w,r)
			return
		}

		next.ServeHTTP(w, r)
	})
}