package controllers

import (
	"net/http"

	"github.com/joncalhoun/lenslocked/context"
)

type Setting struct {
	Templates struct {
		New Template
	}
}

func (s Setting) RenderSetting(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	user := context.User(r.Context())
	data.Email = user.Email
	s.Templates.New.Execute(w, r, data)
}

func (s Setting) ProcessResetPassword(w http.ResponseWriter, r *http.Request) {

}

func (s Setting) ProcessResetEmail(w http.ResponseWriter, r *http.Request) {

}
