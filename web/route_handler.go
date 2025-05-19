package web

import (
	"context"
	"fmt"
	"gofire/internal/repository"
	"gofire/internal/state"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	http.HandleFunc("/", handler.dashboardHandler)
}

func (handler *HttpRouteHandler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pageNumber := getPageNumber(r)
	statusParam := strings.TrimSpace(r.URL.Query().Get("status"))
	status := state.JobStatus(statusParam)

	if err := handler.handleDashboardAction(ctx, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var statuses []state.JobStatus
	if statusParam != "" {
		statuses = append(statuses, status)
	}

	jobs, err := handler.repository.FetchDueJobs(ctx, pageNumber, PageSize, statuses, nil)
	if err != nil {
		log.Printf("failed to fetch jobs: %v", err)
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	allJobsCount, _ := handler.repository.CountAllJobsGroupedByStatus(ctx)

	data := NewPaginatedDataMap(*jobs).
		Add("Statuses", state.AllStatuses).
		Add("CurrentStatus", status).
		Add("JobsCount", allJobsCount)

	render(w, "dashboard", data.Data)
}

func (handler *HttpRouteHandler) handleScheduledJobs() {
	http.HandleFunc("/scheduled", func(w http.ResponseWriter, r *http.Request) {
		render(w, "scheduled", nil)
	})
}

func (handler *HttpRouteHandler) handleDashboardAction(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" {
		return nil
	}

	switch action {
	case "remove":
		id := r.FormValue("id")
		jobID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid job id")
		}
		if err := handler.repository.RemoveByID(ctx, jobID); err != nil {
			return fmt.Errorf("failed to remove job: %w", err)
		}
	default:
		break
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    url.QueryEscape("Job removed successfully!"),
		Path:     "/",
		MaxAge:   5,
		HttpOnly: false,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)

	return nil
}
