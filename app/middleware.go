package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/userservice"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			method = r.Method
			proto  = r.Proto
			uri    = r.URL.RequestURI()
		)

		app.logger.Info("request from", slog.String("method", method), slog.String("uri", uri), slog.String("remote_addr", ip), slog.String("proto", proto))

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Authorization")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			r = app.createUserContext(r, &userservice.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		token := app.extractTokenFromHeader(authHeader)
		if token == "" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := app.userService.GetUserByAccessToken(r.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, common.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			case errors.As(err, &common.ValidationError{}):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.createUserContext(r, user)
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.getUserContext(r)
		if user.IsAnonymous() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivatedUser(next http.Handler) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.getUserContext(r)
		if !user.IsActivated() {
			app.unAuthorizedErrorResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requireAuthUser(fn)
}

func (app *application) requirePermission(next http.HandlerFunc, permission userservice.Permission) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.getUserContext(r)
		if !user.HasPermission(permission) {
			app.unAuthorizedErrorResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requireActivatedUser(fn)
}

// create a caching middleware
func (app *application) cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}
