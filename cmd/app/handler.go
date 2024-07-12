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
	err = app.userService.CreateUser(r.Context(), input.Username, input.Email, input.Password)
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
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "user account created"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type activateUserRequest struct {
	Token string `json:"token"`
}

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

func (app *application) createBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input blogservice.CreateBlogRequest

	// Parse the request body
	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Call the blog service
	err = app.blogService.CreateBlog(r.Context(), &input)
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
	UserID  int    `json:"user_id"`
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

	blog := blogservice.Blog{
		ID:      id,
		Title:   input.Title,
		Content: input.Content,
		UserID:  input.UserID,
	}

	// Call the blog service
	err = app.blogService.UpdateBlog(r.Context(), &blog)
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

type deleteBlogRequest struct {
	UserId int `json:"user_id"`
}

func (app *application) deleteBlogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	var input deleteBlogRequest

	// Parse the request body
	err = app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	// Call the blog service
	err = app.blogService.DeleteBlog(r.Context(), id, input.UserId)
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
