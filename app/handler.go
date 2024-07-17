package main

import (
	"errors"
	"net/http"

	"github.com/sushihentaime/blogist/internal/blogservice"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/userservice"
)

type registerUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input registerUserRequest

	// Parse the request body
	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Call the user service
	token, err := app.userService.CreateUser(r.Context(), input.Username, input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrDuplicateEmail):
			app.failedValidationErrorResponse(w, r, map[string]string{"email": "a user with this email address already exists"})
		case errors.Is(err, userservice.ErrDuplicateUsername):
			app.failedValidationErrorResponse(w, r, map[string]string{"username": "this username is already taken"})
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return the response
	err = app.writeJSON(w, http.StatusCreated, envelope{"token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type activateUserRequest struct {
	Token string `json:"token"`
}

// how to test this with the rabbitmq broker?
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input activateUserRequest

	// Parse the request body
	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Call the user service
	err = app.userService.ActivateUser(r.Context(), input.Token)
	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrNotFound):
			app.notFoundErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "user account activated"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type loginUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (app *application) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var input loginUserRequest

	// Parse the request body
	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Call the user service
	token, err := app.userService.LoginUser(r.Context(), input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrNotFound):
			app.invalidCredentialsErrorResponse(w, r)
		case errors.Is(err, userservice.ErrAuthenticationFailure):
			app.invalidCredentialsErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user from the context
	user := app.getUserContext(r)

	// Call the user service
	err := app.userService.LogoutUser(r.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "user logged out"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type createBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (app *application) createBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input createBlogRequest

	// Parse the request body
	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// get the user from the context
	user := app.getUserContext(r)

	req := &blogservice.CreateBlogRequest{
		Title:   input.Title,
		Content: input.Content,
		UserID:  user.ID,
	}

	// Call the blog service
	err = app.blogService.CreateBlog(r.Context(), req)
	if err != nil {
		switch {
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		case errors.Is(err, blogservice.ErrUserForeignKey):
			app.unAuthorizedErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "blog created"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getBlogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"blog": blog}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type updateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (app *application) updateBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input updateBlogRequest

	// id is a URL parameter
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Parse the request body
	err = app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	user := app.getUserContext(r)

	// get the blog from the database
	dbBlog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if dbBlog.User.ID != user.ID {
		app.unAuthorizedErrorResponse(w, r)
		return
	}

	// Call the blog service
	err = app.blogService.UpdateBlog(r.Context(), input.Title, input.Content, &dbBlog.ID, &user.ID, &dbBlog.Version)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "blog updated"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteBlogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	dbBlog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user := app.getUserContext(r)

	if dbBlog.User.ID != user.ID {
		app.unAuthorizedErrorResponse(w, r)
		return
	}

	// Call the blog service
	err = app.blogService.DeleteBlog(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "blog deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getAllBlogsHandler(w http.ResponseWriter, r *http.Request) {
	// get the limit and offset query parameters
	limit, offset, err := app.readLimitOffsetParams(r)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blogs, err := app.blogService.GetBlogs(r.Context(), limit, offset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"blogs": blogs}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) searchBlogsHandler(w http.ResponseWriter, r *http.Request) {
	title, err := app.readStringParam(r, "q")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// get the limit and offset query parameters
	limit, offset, err := app.readLimitOffsetParams(r)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blogs, err := app.blogService.GetBlogsByTitle(r.Context(), title, limit, offset)
	if err != nil {
		switch {
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"blogs": blogs}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getBlogsByUserIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "userid")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blogs, err := app.blogService.GetBlogsByUserId(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, blogservice.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"blogs": blogs}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
