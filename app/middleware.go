package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/userservice"
	"golang.org/x/time/rate"
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
		w.Header().Add("Vary", "Authorization")

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

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The Vary header is used to tell the browser that the response can vary depending on the value of the Origin header. This is useful when the server is serving different responses based on the origin of the request. The Vary header tells the browser to cache the response based on the value of the Origin header.
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// An origin consists of 3 components: protocol, host and port. When a request is made from a browser http:www.example.com, the Origin header is included in the request. However, when the request is made from a ip address, the Origin header is not included in the request. This is because the ip address does not have a host component.

		// When the Origin header is included in the request, the server should check if the origin is in the list of trusted origins. If the origin is in the list of trusted origins, the server should include the Access-Control-Allow-Origin header in the response with the value of the origin. This allows the browser to make the request.

		// For allowing preflight requests, the server should check if the request method is OPTIONS and the Access-Control-Request-Method header is not empty. If the conditions are met, the server should include the Access-Control-Allow-Methods and Access-Control-Allow-Headers headers in the response. The server should also set the status code to 200.
		origin := r.Header.Get("Origin")

		if origin != "" {
			for i := range app.config.TrustedOrigins {
				if origin == app.config.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
						w.WriteHeader(http.StatusOK)
						return
					}

					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.config.RateLimitEnabled {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.RateLimitRPS), app.config.RateLimitBurst)}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
