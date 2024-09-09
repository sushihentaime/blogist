package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/sushihentaime/blogist/internal/userservice"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundErrorResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedErrorResponse)

	router.HandlerFunc(http.MethodGet, "/swagger/*any", httpSwagger.WrapHandler.ServeHTTP)

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

	router.HandlerFunc(http.MethodGet, "/api/v1/blogs/view/:id", app.getBlogHandler)
	router.HandlerFunc(http.MethodPut, "/api/v1/blogs/update/:id", app.requirePermission(app.updateBlogHandler, userservice.PermissionWriteBlog))
	router.HandlerFunc(http.MethodDelete, "/api/v1/blogs/delete/:id", app.requirePermission(app.deleteBlogHandler, userservice.PermissionWriteBlog))

	router.HandlerFunc(http.MethodPost, "/api/v1/blogs/like/:id", app.requireActivatedUser(app.likeBlogHandler))
	router.HandlerFunc(http.MethodPut, "/api/v1/blogs/unlike/:id", app.requireActivatedUser(app.unlikeBlogHandler))

	router.HandlerFunc(http.MethodGet, "/api/v1/blogs/user/:userid", app.getBlogsByUserIdHandler)

	// Add a metrics handler
	router.HandlerFunc(http.MethodGet, "/debugs/vars", expvar.Handler().ServeHTTP)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.logRequest(app.authenticate(router)))))
}
