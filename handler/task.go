package handler

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "tasks-api/store"
)

type TaskHandler struct {
    store *store.TaskStore
}

func NewTaskHandler(s *store.TaskStore) *TaskHandler {
    return &TaskHandler{store: s}
}

type createRequest struct {
    Title string `json:"title"`
}

type updateRequest struct {
    Title string `json:"title"`
    Done  bool   `json:"done"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, h.store.GetAll())
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    task, err := h.store.GetByID(id)
    if err != nil {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req createRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
        return
    }
    task := h.store.Create(req.Title)
    writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var req updateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
        return
    }
    task, err := h.store.Update(id, req.Title, req.Done)
    if err != nil {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := h.store.Delete(id); err != nil {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
        return
    }
    w.WriteHeader(http.StatusNoContent)
}