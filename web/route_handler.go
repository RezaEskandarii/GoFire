package web

import (
	"fmt"
	"gofire/internal/repository"
	"gofire/internal/state"
	"log"
	"net/http"
)

const (
	PageSize = 15
)

type HttpRouteHandler struct {
	repository repository.EnqueuedJobRepository
}

func NewRouteHandler(repository repository.EnqueuedJobRepository) HttpRouteHandler {
	return HttpRouteHandler{
		repository: repository,
	}
}

func (handler *HttpRouteHandler) Serve(port int) {
	handler.handleDashboard()
	handler.handleScheduledJobs()

	addr := fmt.Sprintf(":%d", port)
	printBanner(addr)
	http.ListenAndServe(addr, nil)
}

func (handler *HttpRouteHandler) handleDashboard() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pageNumber := getPageNumber(r)
		statusParam := r.URL.Query().Get("status")
		status := state.JobStatus(statusParam)

		jobs, err := handler.repository.FetchDueJobs(r.Context(), pageNumber, PageSize, &status, nil)
		if err != nil {
			log.Println(err)
		}

		allJobsCount
		data := NewPaginatedDataMap(*jobs).
			Add("Statuses", state.AllStatuses).
			Add("CurrentStatus", status)

		render(w, "dashboard", data.Data)
	})
}

func (handler *HttpRouteHandler) handleScheduledJobs() {
	http.HandleFunc("/scheduled", func(w http.ResponseWriter, r *http.Request) {
		render(w, "scheduled", nil)
	})
}
