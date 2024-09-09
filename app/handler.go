package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/sushihentaime/blogist/internal/blogservice"
	"github.com/sushihentaime/blogist/internal/common"
	"github.com/sushihentaime/blogist/internal/likeservice"
	"github.com/sushihentaime/blogist/internal/userservice"
)

type registerUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

// @Summary Register a new user
// @Description Register a new user with a username, email, and password
// @Tags Users
// @Accept json
// @Produce json
// @Param input body registerUserRequest true "User registration details"
// @Success 201 {object} TokenResponse "Successfully registered user"
// @Router /users/register [post]
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input registerUserRequest

	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

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

	err = app.writeJSON(w, http.StatusCreated, envelope{"token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type activateUserRequest struct {
	Token string `json:"token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

// @Summary Activate a user account
// @Description Activate a user account using the provided token
// @Tags Users
// @Accept json
// @Produce json
// @Param input body activateUserRequest true "User activation details"
// @Success 200 {object} MessageResponse "User account activated"
// @Router /users/activate [put]
// ! how to test this with the rabbitmq broker?
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input activateUserRequest

	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	err = app.userService.ActivateUser(r.Context(), input.Token)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

type AuthTokenResponse struct {
	Token struct {
		AccessToken        string    `json:"access_token"`
		RefreshToken       string    `json:"refresh_token"`
		UserID             int       `json:"user_id"`
		AccessTokenExpiry  time.Time `json:"access_token_expiry"`
		RefreshTokenExpiry time.Time `json:"refresh_token_expiry"`
	} `json:"token"`
}

// @Summary Log in a user
// @Description Log in a user using their username and password
// @Tags Users
// @Accept json
// @Produce json
// @Param input body loginUserRequest true "User login details"
// @Success 200 {object} AuthTokenResponse "Successfully logged in user"
// @Router /users/login [post]
func (app *application) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var input loginUserRequest

	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	token, err := app.userService.LoginUser(r.Context(), input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

// @Summary Log out a user
// @Description Log out a user by invalidating their access token
// @Tags Users
// @Security Bearer
// @Accept json
// @Produce json
// @Success 200 {object} MessageResponse "User logged out"
// @Router /users/logout [delete]
func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getUserContext(r)

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

// @Summary Create a new blog
// @Description Create a new blog with a title and content, authenticated by a logged-in user
// @Tags Blogs
// @Security Bearer
// @Accept json
// @Produce json
// @Param input body createBlogRequest true "Blog creation details"
// @Success 201 {object} BlogResponse "Blog created"
// @Router /blogs/create [post]
func (app *application) createBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input createBlogRequest

	err := app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	user := app.getUserContext(r)

	req := &blogservice.CreateBlogRequest{
		Title:   input.Title,
		Content: input.Content,
		UserID:  user.ID,
	}

	blog, err := app.blogService.CreateBlog(r.Context(), req)
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

	err = app.writeJSON(w, http.StatusCreated, envelope{"blog": blog}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type BlogResponse struct {
	Blog      blogservice.Blog        `json:"blog"`
	LikedBy   []likeservice.LikedUser `json:"liked_by"`
	LikeCount int                     `json:"like_count"`
}

// @Summary Get a blog by ID
// @Description Retrieve a specific blog post by its ID
// @Tags Blogs
// @Param id path int true "Blog ID"
// @Produce json
// @Success 200 {object} BlogResponse "Blog post"
// @Router /blogs/view/{id} [get]
func (app *application) getBlogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
			app.notFoundErrorResponse(w, r)
		case errors.As(err, &common.ValidationError{}):
			validationErr := err.(common.ValidationError)
			app.failedValidationErrorResponse(w, r, validationErr.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	users, likeCount, err := app.fetchBlogAndLikesData(r.Context(), blog.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	response := envelope{
		"blog":       blog,
		"liked_by":   users,
		"like_count": likeCount,
	}

	err = app.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

type updateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// @Summary Update an existing blog
// @Description Update an existing blog post by its ID, authenticated by the blog's author. Only the author can update their own blog.
// @Tags Blogs
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "Blog ID"
// @Param input body updateBlogRequest true "Blog update details"
// @Success 200 {object} MessageResponse "Blog updated"
// @Router /blogs/update/{id} [put]
func (app *application) updateBlogHandler(w http.ResponseWriter, r *http.Request) {
	var input updateBlogRequest

	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	err = app.parseJSON(w, r, &input)
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	user := app.getUserContext(r)

	dbBlog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

	err = app.blogService.UpdateBlog(r.Context(), input.Title, input.Content, &dbBlog.ID, &user.ID, &dbBlog.Version)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

// @Summary Delete a blog
// @Description Delete a blog post by its ID, authenticated by the blog's author. Only the author can delete their own blog.
// @Tags Blogs
// @Security Bearer
// @Param id path int true "Blog ID"
// @Router /blogs/delete/{id} [delete]
func (app *application) deleteBlogHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	dbBlog, err := app.blogService.GetBlogByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

	err = app.blogService.DeleteBlog(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

type AllBlogsResponse struct {
	Blogs []blogservice.Blog `json:"blogs"`
}

// @Summary Get all blogs
// @Description Retrieve a paginated list of all blog posts
// @Tags Blogs
// @Param limit query int false "The number of blogs to return"
// @Param offset query int false "The number of blogs to skip"
// @Produce json
// @Success 200 {object} AllBlogsResponse "List of blog posts"
// @Router /blogs [get]
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

// @Summary Search for blogs by title
// @Description Retrieve a paginated list of blog posts that match the given title query.
// @Tags Blogs
// @Produce json
// @Param q query string true "The title query to search for"
// @Param limit query int false "The number of blogs to return"
// @Param offset query int false "The number of blogs to skip"
// @Success 200 {object} AllBlogsResponse "List of blog posts"
// @Router /blogs/search [get]
func (app *application) searchBlogsHandler(w http.ResponseWriter, r *http.Request) {
	title, err := app.readStringParam(r, "q")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

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

// @Summary Retrieve blogs by user ID
// @Description Get a list of blog posts authored by a specific user
// @Tags Blogs
// @Param userid path int true "User ID"
// @Produce json
// @Success 200 {object} AllBlogsResponse "List of blog posts"
// @Router /blogs/user/{userid} [get]
func (app *application) getBlogsByUserIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "userid")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	blogs, err := app.blogService.GetBlogsByUserId(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, common.ErrRecordNotFound):
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

// @Summary Like a blog post
// @Description Like a blog post by its ID, authenticated by a logged-in user
// @Tags Blogs
// @Security Bearer
// @Param id path int true "Blog ID"
// @Produce json
// @Success 201 {object} MessageResponse "Blog liked"
// @Router /blogs/like/{id} [post]
func (app *application) likeBlogHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getUserContext(r)

	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	err = app.likeService.CreateLike(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, likeservice.ErrInvalidBlogID) || errors.Is(err, likeservice.ErrInvalidUserID):
			app.notFoundErrorResponse(w, r)
		case errors.Is(err, likeservice.ErrAlreadyLiked):
			app.writeJSON(w, http.StatusOK, envelope{"message": "blog already liked"}, nil)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "Blog liked successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// @Summary Unlike a blog post
// @Description Unlike a blog post by its ID, authenticated by a logged-in user
// @Tags Blogs
// @Security Bearer
// @Param id path int true "Blog ID"
// @Produce json
// @Success 200 {object} MessageResponse "Blog unliked successfully"
// @Router /blogs/unlike/{id} [put]
func (app *application) unlikeBlogHandler(w http.ResponseWriter, r *http.Request) {
	user := app.getUserContext(r)

	id, err := app.readIDParam(r, "id")
	if err != nil {
		app.badRequestErrorResponse(w, r, err)
		return
	}

	err = app.likeService.DeleteLike(r.Context(), id, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, likeservice.ErrInvalidBlogID) || errors.Is(err, likeservice.ErrInvalidUserID):
			app.notFoundErrorResponse(w, r)
		case errors.Is(err, likeservice.ErrLikeNotFound):
			app.notFoundErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "Blog unliked successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
