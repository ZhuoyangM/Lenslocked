package views

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/csrf"
	"github.com/joncalhoun/lenslocked/context"
	"github.com/joncalhoun/lenslocked/models"
)

type Template struct {
	htmlTpl *template.Template
}

// We will use this to determine if an error provides the Public method.
type publicError interface {
	Public() string
}

// Parse and return the template
func ParseFS(fs fs.FS, patterns ...string) (Template, error) {
	tpl := template.New(filepath.Base(patterns[0]))

	//update the template's function map
	tpl = tpl.Funcs(
		template.FuncMap{
			"csrfField": func() (template.HTML, error) {
				return "", fmt.Errorf("csrfField not implemented")
			}, // placeholder
			"currentUser": func() (*models.User, error) {
				return nil, fmt.Errorf("currentUser not implemented")
			},
			"errors": func() []string {
				return nil
			},
		},
	)

	//Parse the template with the new function field
	tpl, err := tpl.ParseFS(fs, patterns...)
	if err != nil {
		return Template{}, fmt.Errorf("Error parsing template: %w", err)
	}

	return Template{
		htmlTpl: tpl,
	}, nil
}

// Execute the template
func (t Template) Execute(w http.ResponseWriter, r *http.Request, data interface{}, errs ...error) {
	//Clone the template so each web request will have their own clone (Concurrency safe)
	tpl, err := t.htmlTpl.Clone()
	if err != nil {
		log.Printf("Error cloning the template: %v", err)
		http.Error(w, "Error rendering the page", http.StatusInternalServerError)
		return
	}

	errMsgs := errMessages(errs...)
	tpl = tpl.Funcs(
		template.FuncMap{
			"csrfField": func() template.HTML {
				return csrf.TemplateField(r) //HTML tag that provides the csrf token
			},
			"currentUser": func() *models.User {
				return context.User(r.Context())
			},
			"errors": func() []string {
				return errMsgs
			},
		},
	)

	w.Header().Set("Content-Type", "text/html; chartset=utf-8")
	err = tpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func Must(t Template, err error) Template {
	if err != nil {
		panic(err)
	}
	return t
}

// Return all error messages
func errMessages(errs ...error) []string {
	var msgs []string
	for _, err := range errs {
		var pubErr publicError
		if errors.As(err, &pubErr) {
			msgs = append(msgs, pubErr.Public())
		} else {
			fmt.Println(err)
			msgs = append(msgs, "Something went wrong.")
		}
	}
	return msgs
}
