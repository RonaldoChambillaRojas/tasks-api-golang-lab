package store

import (
    "errors"
    "sync"
	"fmt"
    "tasks-api/model"
)

type TaskStore struct {
    mu      sync.RWMutex
    tasks   map[string]model.Task
    counter int
}

func NewTaskStore() *TaskStore {
    return &TaskStore{
        tasks: make(map[string]model.Task),
    }
}

func (s *TaskStore) GetAll() []model.Task {
    s.mu.RLock()
    defer s.mu.RUnlock()

    list := make([]model.Task, 0, len(s.tasks))
    for _, t := range s.tasks {
        list = append(list, t)
    }
    return list
}

func (s *TaskStore) GetByID(id string) (model.Task, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    t, ok := s.tasks[id]
    if !ok {
        return model.Task{}, errors.New("task not found")
    }
    return t, nil
}

func (s *TaskStore) Create(title string) model.Task {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.counter++
    id := fmt.Sprintf("%d", s.counter)
    t := model.Task{ID: id, Title: title, Done: false}
    s.tasks[id] = t
    return t
}

func (s *TaskStore) Update(id string, title string, done bool) (model.Task, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    t, ok := s.tasks[id]
    if !ok {
        return model.Task{}, errors.New("task not found")
    }
    t.Title = title
    t.Done = done
    s.tasks[id] = t
    return t, nil
}

func (s *TaskStore) Delete(id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, ok := s.tasks[id]; !ok {
        return errors.New("task not found")
    }
    delete(s.tasks, id)
    return nil
}