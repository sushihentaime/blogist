package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/sushihentaime/blogist/internal/userservice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundErrorResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedErrorResponse)

	// health check
	router.HandlerFunc(http.MethodGet, "/health", app.healthCheckHandler)

	// user service
	router.HandlerFunc(http.MethodPost, "/api/v1/users/register", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/api/v1/users/activate", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/api/v1/users/login", app.loginUserHandler)
	router.HandlerFunc(http.MethodDelete, "/api/v1/users/logout", app.logoutUserHandler)

	// blog service
	router.HandlerFunc(http.MethodGet, "/api/v1/blogs", app.getAllBlogsHandler)
	router.HandlerFunc(http.MethodPost, "/api/v1/blogs/create", app.requirePermission(app.createBlogHandler, userservice.PermissionWriteBlog))
	router.HandlerFunc(http.MethodGet, "/api/v1/blogs/search", app.searchBlogsHandler)
	router.HandlerFunc(http.MethodGet, "/api/v1/blogs/user/:userid", app.getBlogsByUserIdHandler)
	router.HandlerFunc(http.MethodGet, "/api/v1/blogs/view/:id", app.getBlogHandler)
	router.HandlerFunc(http.MethodPut, "/api/v1/blogs/update/:id", app.requirePermission(app.updateBlogHandler, userservice.PermissionWriteBlog))
	router.HandlerFunc(http.MethodDelete, "/api/v1/blogs/delete/:id", app.requirePermission(app.deleteBlogHandler, userservice.PermissionWriteBlog))

	return app.recoverPanic(app.logRequest(app.authenticate(router)))
}
