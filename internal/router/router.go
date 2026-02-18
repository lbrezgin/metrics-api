// Package router configures the HTTP routes for the service.
//
// It is responsible only for registering routes and middleware and wiring
// HTTP endpoints to handler methods. Request-handling and business logic
// live in the handler and service layers.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type handler interface {
	Update(w http.ResponseWriter, r *http.Request)
	List(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	UpdateFromBody(w http.ResponseWriter, r *http.Request)
	ShowFromBody(w http.ResponseWriter, r *http.Request)
}

// New returns a configured chi router. It accepts a value that implements
// the handler interface and middlewares.
func New(h handler, mws ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(mws...)

	r.Get("/", h.List)
	r.Get("/value/{type}/{name}", h.Show)

	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update", h.UpdateFromBody)
	r.Post("/value", h.ShowFromBody)
	return r
}
