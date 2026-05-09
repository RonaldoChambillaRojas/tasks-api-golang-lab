package main

import (
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "tasks-api/handler"
    "tasks-api/store"
)

func main() {
    s := store.NewTaskStore()
    h := handler.NewTaskHandler(s)

    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    r.Get("/tasks", h.GetAll)
    r.Post("/tasks", h.Create)
    r.Get("/tasks/{id}", h.GetByID)
    r.Put("/tasks/{id}", h.Update)
    r.Delete("/tasks/{id}", h.Delete)

    log.Println("Server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}