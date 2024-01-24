package controllers

import (
	"fmt"
	"net/http"
)

const (
	CookieSession = "session"
)

// Create a new cookie and return it
func newCookie(name, value string) *http.Cookie {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
	}
	return &cookie
}

// Create and set cookie
func setCookie(w http.ResponseWriter, name, value string) {
	cookie := newCookie(name, value)
	http.SetCookie(w, cookie)
}

// Read the cookie value with given name
func readCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", fmt.Errorf("Error reading cookie %s: %w", name, err)
	}
	return cookie.Value, nil
}

// Delete the cookie with given name
func deleteCookie(w http.ResponseWriter, name string) {
	cookie := newCookie(name, "")
	cookie.MaxAge = -1 //Invalidate the cookie (delete)
	http.SetCookie(w, cookie)
}
