package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundErrorResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedErrorResponse)

	router.HandlerFunc(http.MethodPost, "/v1/users/register", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activate", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/users/login", app.loginUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/blogs", app.createBlogHandler)
	router.HandlerFunc(http.MethodGet, "/v1/blogs/:id", app.getBlogHandler)
	router.HandlerFunc(http.MethodPut, "/v1/blogs/:id", app.updateBlogHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/blogs/:id", app.deleteBlogHandler)
	router.HandlerFunc(http.MethodGet, "/v1/blogs", app.getAllBlogsHandler)
	router.HandlerFunc(http.MethodGet, "/v1/blogs/search", app.searchBlogsHandler)
	router.HandlerFunc(http.MethodGet, "/v1/blogs/user/:id", app.getBlogsByUserIdHandler)

	return router
}
