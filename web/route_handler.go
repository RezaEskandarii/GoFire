package web

import (
	"fmt"
	"gofire/internal/db"
	"gofire/internal/state"
	"log"
	"net/http"
	"strconv"
)

const (
	PageSize = 15
)

type HttpRouteHandler struct {
	repository db.EnqueuedJobRepository
}

func NewRouteHandler(repository db.EnqueuedJobRepository) HttpRouteHandler {
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

		jobs, err := handler.repository.FetchDueJobs(r.Context(), getPageNumber(r), PageSize, &status, nil)
		if err != nil {
			log.Println(err)
		}
		data := map[string]interface{}{
			"Page":            pageNumber,
			"TotalPages":      jobs.TotalPages,
			"Items":           jobs.Items,
			"HasPreviousPage": jobs.HasPreviousPage,
			"HasNextPage":     jobs.HasNextPage,
			"TotalItems":      jobs.TotalItems,
			"Statuses":        state.AllStatuses,
			"CurrentStatus":   status,
		}
		render(w, "dashboard", data)
	})
}

func (handler *HttpRouteHandler) handleScheduledJobs() {
	http.HandleFunc("/scheduled", func(w http.ResponseWriter, r *http.Request) {
		render(w, "scheduled", nil)
	})
}

func getPageNumber(r *http.Request) int {
	page := r.URL.Query().Get("page")
	pageNumber, err := strconv.ParseInt(page, 10, 64)
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}
	return int(pageNumber)
}

func printBanner(addr string) {
	width := 46
	fmt.Println("##############################################")
	fmt.Printf("# %-*s #\n", width-4, "")
	fmt.Printf("# %-*s #\n", width-4, "GoFire Started")
	fmt.Printf("# %-*s #\n", width-4, fmt.Sprintf("GoFire Server running on %s", addr))
	fmt.Printf("# %-*s #\n", width-4, "")
	fmt.Println("##############################################")
}
