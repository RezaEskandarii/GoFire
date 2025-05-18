package web

import (
	"fmt"
	"gofire/internal/db"
	"net/http"
	"strings"
)

type HttpRouteHandler struct {
	repository db.EnqueuedJobRepository
}

func NewRouteHandler(repository db.EnqueuedJobRepository) HttpRouteHandler {
	return HttpRouteHandler{
		repository: repository,
	}
}

func (handler *HttpRouteHandler) Serve(port string) {
	handler.handleDashboard()
	handler.handleScheduledJobs()

	if strings.TrimSpace(port) == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func (handler *HttpRouteHandler) handleDashboard() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{}
		render(w, "dashboard", data)
	})
}

func (handler *HttpRouteHandler) handleScheduledJobs() {
	http.HandleFunc("/scheduled", func(w http.ResponseWriter, r *http.Request) {
		render(w, "scheduled", nil)
	})
}
