package controllers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/joncalhoun/lenslocked/context"
	"github.com/joncalhoun/lenslocked/errors"
	"github.com/joncalhoun/lenslocked/models"
)

// Users type for user controllers
type Users struct {
	Templates struct {
		SignUp                Template
		SignIn                Template
		ForgotPassword        Template
		CheckYourEmail        Template
		ResetPassword         Template
		Setting               Template
		EmailUpdateSuccess    Template
		PasswordChangeSuccess Template
	}
	UserService          *models.UserService
	SessionService       *models.SessionService
	PasswordResetService *models.PasswordResetService
	EmailService         *models.EmailService
	EmailResetService    *models.EmailResetService
}

// User middleware
type UserMiddleWare struct {
	SessionService *models.SessionService
}

// Render sign up form
func (u Users) SignUp(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.SignUp.Execute(w, r, data)
}

// Render sign in form
func (u Users) SignIn(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.SignIn.Execute(w, r, data)
}

// Process sign up requests
func (u Users) ProcessSignUp(w http.ResponseWriter, r *http.Request) {
	//Create a new user
	var data struct {
		Email    string
		Password string
	}
	data.Email = r.FormValue("email")
	data.Password = r.FormValue("password")
	user, err := u.UserService.Create(data.Email, data.Password)
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			err = errors.Public(err, "That email address is already associated with an account.")
		}
		u.Templates.SignUp.Execute(w, r, data, err)
		return
	}

	//Create a new session for the user
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		// TODO: Show a warning message about not being able to sign the user in
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}

	//Set session cookie and redirect to user's home page
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

// Process sign in requests
func (u Users) ProcessSignIn(w http.ResponseWriter, r *http.Request) {
	//Authenticate the user
	var data struct {
		Email    string
		Password string
	}
	data.Email = r.FormValue("email")
	data.Password = r.FormValue("password")
	user, err := u.UserService.Authenticate(data.Email, data.Password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	//Create a session for the user
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	//Set session cookie and redirect to user's homepage
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

// Render current user's home page.
// SetUser and RequiredUser middlewares are necessary.
func (u Users) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user := context.User(r.Context())
	fmt.Fprintf(w, "Current user: %s\n", user.Email)
}

// Sign out the user
func (u Users) ProcessSignOut(w http.ResponseWriter, r *http.Request) {
	// Read the session token
	token, err := readCookie(r, CookieSession)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	//Delete session from database
	err = u.SessionService.Delete(token)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	// Delete session cookie
	deleteCookie(w, CookieSession)
	http.Redirect(w, r, "/signin", http.StatusFound)
}

// Set user in request context
func (userMw UserMiddleWare) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := readCookie(r, CookieSession)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := userMw.SessionService.User(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		ctx = context.WithUser(ctx, user)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// Require a user to be present in the web request
func (userMw UserMiddleWare) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := context.User(r.Context())
		if user == nil {
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Render the forgot password page
func (u Users) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.ForgotPassword.Execute(w, r, data)
}

// Process the forgot password page
func (u Users) ProcessForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	pwReset, err := u.PasswordResetService.Create(data.Email)
	if err != nil {
		// TODO: Handle other cases in the future. For instance,
		// if a user doesn't exist with the email address.
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	vals := url.Values{
		"token": {pwReset.Token},
	}
	// TODO: Make the URL here configurable
	resetURL := "https://www.lenslocked.com/reset-pw?" + vals.Encode()
	err = u.EmailService.ForgotPassword(data.Email, resetURL)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	u.Templates.CheckYourEmail.Execute(w, r, data)
}

// Render reset password form
func (u Users) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string
	}
	data.Token = r.FormValue("token")
	u.Templates.ResetPassword.Execute(w, r, data)
}

// Process reset password form
func (u Users) ProcessResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token    string
		Password string
	}
	data.Token = r.FormValue("token")
	data.Password = r.FormValue("password")

	user, err := u.PasswordResetService.Consume(data.Token)
	if err != nil {
		fmt.Println(err)
		// TODO: Distinguish between server errors and invalid token errors.
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Update the user's password.
	err = u.UserService.UpdatePassword(user.ID, data.Password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Sign the user in now that they have reset their password.
	// Any errors from this point onward should redirect to the sign in page.
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/galleries", http.StatusFound)
}

// Render setting's page
func (u Users) RenderSetting(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	user := context.User(r.Context())
	data.Email = user.Email
	u.Templates.Setting.Execute(w, r, data)
}

// Process password update when signed in
func (u Users) ProcessUpdatePassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Password string
	}
	data.Password = r.FormValue("password")
	user := context.User(r.Context())
	err := u.UserService.UpdatePassword(user.ID, data.Password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	u.Templates.PasswordChangeSuccess.Execute(w, r, data)
}

// Send email-reset email to the new email account and redirect to a email-sent confirmation page
func (u Users) ProcessUpdateEmail(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	user := context.User(r.Context())
	emailReset, err := u.EmailResetService.Create(user.ID)
	if err != nil {
		// TODO: Handle other cases in the future. For instance,
		// if a user doesn't exist with the email address.
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	vals := url.Values{
		"token": {emailReset.Token},
		"email": {data.Email},
	}
	// TODO: Make the URL here configurable
	resetURL := "https://www.lenslocked.com/setting/reset-email?" + vals.Encode()
	err = u.EmailService.SendUpdateEmail(data.Email, resetURL)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	u.Templates.CheckYourEmail.Execute(w, r, data)

}

// Process email reset after user clicks on the reset link
func (u Users) ProcessResetEmail(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string
		Email string
	}
	data.Token = r.FormValue("token")
	data.Email = r.FormValue("email")

	user, err := u.EmailResetService.Consume(data.Token)
	if err != nil {
		fmt.Println(err)
		// TODO: Distinguish between server errors and invalid token errors.
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Update the user's email
	err = u.UserService.UpdateEmail(user.ID, data.Email)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	u.Templates.EmailUpdateSuccess.Execute(w, r, data)
}
