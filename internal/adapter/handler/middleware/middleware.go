package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/godlew/homecoin/internal/adapter/handler/response"
	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
)

type contextKey string

const UserIDKey contextKey = "userID"

func Auth(jwt *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Error(w, domainerrors.ErrUnauthorized)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			userID, err := jwt.ParseAccessToken(token)
			if err != nil {
				response.Error(w, domainerrors.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				response.Error(w, domainerrors.ErrInvalidInput)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
