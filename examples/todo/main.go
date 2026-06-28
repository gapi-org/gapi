package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gapi-org/gapi"
	"github.com/gapi-org/gapi/middleware"
)

type Todo struct {
	ID        string    `json:"id" doc:"Todo ID"`
	Title     string    `json:"title" validate:"required,min=1,max=120"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at" format:"date-time"`
}

type ListTodosOut struct {
	Todos []Todo `json:"todos"`
}

type CreateTodoIn struct {
	Body struct {
		Title string `json:"title" validate:"required,min=1,max=120"`
	} `body:""`
}

type GetTodoIn struct {
	ID string `path:"id" validate:"required"`
}

type UpdateTodoIn struct {
	ID   string `path:"id" validate:"required"`
	Body struct {
		Title     string `json:"title" validate:"min=1,max=120"`
		Completed bool   `json:"completed"`
	} `body:""`
}

type DeleteTodoOut struct {
	Deleted bool `json:"deleted"`
}

var (
	mu     sync.Mutex
	nextID = 1
	todos  = map[string]Todo{}
)

func Health(ctx context.Context, in struct{}) (map[string]string, error) {
	return map[string]string{"status": "ok"}, nil
}

func ListTodos(ctx context.Context, in struct{}) (ListTodosOut, error) {
	mu.Lock()
	defer mu.Unlock()

	out := ListTodosOut{Todos: make([]Todo, 0, len(todos))}
	for _, todo := range todos {
		out.Todos = append(out.Todos, todo)
	}
	return out, nil
}

func CreateTodo(ctx context.Context, in CreateTodoIn) (gapi.Response[Todo], error) {
	mu.Lock()
	defer mu.Unlock()

	id := fmt.Sprintf("todo-%d", nextID)
	nextID++
	todo := Todo{
		ID:        id,
		Title:     in.Body.Title,
		CreatedAt: time.Now().UTC(),
	}
	todos[id] = todo

	return gapi.Response[Todo]{
		Status: http.StatusCreated,
		Headers: http.Header{
			"Location": []string{"/api/v1/todos/" + id},
		},
		Body: todo,
	}, nil
}

func GetTodo(ctx context.Context, in GetTodoIn) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	todo, ok := todos[in.ID]
	if !ok {
		return Todo{}, errors.New("todo not found")
	}
	return todo, nil
}

func UpdateTodo(ctx context.Context, in UpdateTodoIn) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	todo, ok := todos[in.ID]
	if !ok {
		return Todo{}, errors.New("todo not found")
	}
	if in.Body.Title != "" {
		todo.Title = in.Body.Title
	}
	todo.Completed = in.Body.Completed
	todos[in.ID] = todo
	return todo, nil
}

func DeleteTodo(ctx context.Context, in GetTodoIn) (DeleteTodoOut, error) {
	mu.Lock()
	defer mu.Unlock()

	delete(todos, in.ID)
	return DeleteTodoOut{Deleted: true}, nil
}

func main() {
	app := gapi.New(gapi.Config{
		Title:   "Todo API",
		Version: "0.3.0",
	})
	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())

	gapi.Get[struct{}, map[string]string](app, "/health", Health, gapi.Summary("Health check"))

	api := app.Group("/api/v1")
	gapi.Get[struct{}, ListTodosOut](api, "/todos", ListTodos, gapi.Summary("List todos"), gapi.Tags("todos"))
	gapi.Post[CreateTodoIn, gapi.Response[Todo]](api, "/todos", CreateTodo, gapi.Summary("Create todo"), gapi.Tags("todos"))
	gapi.Get[GetTodoIn, Todo](api, "/todos/{id}", GetTodo, gapi.Summary("Get todo"), gapi.Tags("todos"))
	gapi.Patch[UpdateTodoIn, Todo](api, "/todos/{id}", UpdateTodo, gapi.Summary("Update todo"), gapi.Tags("todos"))
	gapi.Delete[GetTodoIn, DeleteTodoOut](api, "/todos/{id}", DeleteTodo, gapi.Summary("Delete todo"), gapi.Tags("todos"))

	http.ListenAndServe(":8080", app)
}
