package main

import (
	"log/slog"
	"net/http"
)

func (app *application) logError(r *http.Request, err error) {
	var (
		method  = r.Method
		url     = r.URL.RequestURI()
		message = err.Error()
	)

	app.logger.Error(message, slog.String("method", method), slog.String("url", url))
}

func (app *application) writeErrorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	err := app.writeJSON(w, status, envelope{"error": message}, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.writeErrorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) badRequestErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.writeErrorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) notFoundErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.writeErrorResponse(w, r, http.StatusNotFound, "resource not found")
}

func (app *application) failedValidationErrorResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.writeErrorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (app *application) invalidCredentialsErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.writeErrorResponse(w, r, http.StatusUnauthorized, "invalid authentication credentials")
}

func (app *application) unAuthorizedErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.writeErrorResponse(w, r, http.StatusUnauthorized, "unauthorized access")
}

func (app *application) methodNotAllowedErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.writeErrorResponse(w, r, http.StatusMethodNotAllowed, "method not allowed")
}
