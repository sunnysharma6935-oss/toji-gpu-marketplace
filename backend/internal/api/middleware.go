package api

import (
	"context"
	"net/http"
	"strings"

	"gpumarketplace/backend/internal/auth"
)

type ctxKey string

const ctxUserID ctxKey = "user_id"

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")

		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "missing auth token", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(h, "Bearer ")

		userID, err := auth.ParseToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}