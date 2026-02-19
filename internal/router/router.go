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
	Ping(w http.ResponseWriter, r *http.Request)
	Updates(w http.ResponseWriter, r *http.Request)
}

func New(h handler, mws ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(mws...)

	r.Get("/", h.List)
	r.Get("/value/{type}/{name}", h.Show)
	r.Get("/ping", h.Ping)

	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update", h.UpdateFromBody)
	r.Post("/value", h.ShowFromBody)
	r.Post("/updates", h.Updates)
	return r
}
