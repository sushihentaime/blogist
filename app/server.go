package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve(port string) error {

	srv := &http.Server{
		Addr:         port,
		Handler:      app.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.Info("shutting down server", slog.String("signal", s.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		shutdownError <- nil

	}()

	app.logger.Info("starting server", slog.String("port", port), slog.String("env", app.config.Environment))

	if app.config.Environment == "production" {
		err := srv.ListenAndServeTLS(app.config.TLSCertFile, app.config.TLSKeyFile)
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	} else {
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	err := <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", slog.String("addr", srv.Addr))

	return nil
}
