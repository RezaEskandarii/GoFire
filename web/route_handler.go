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
	enqueuedJobRepository repository.EnqueuedJobRepository
	userRepository        repository.UserRepository
}

func NewRouteHandler(repository repository.EnqueuedJobRepository, userRepository repository.UserRepository) HttpRouteHandler {
	return HttpRouteHandler{
		enqueuedJobRepository: repository,
		userRepository:        userRepository,
	}
}

func (handler *HttpRouteHandler) Serve(useAuth bool, port int) {
	handler.handleDashboard(useAuth)
	handler.handleScheduledJobs()
	handler.handleLogin()
	handler.handleLogout()
	addr := fmt.Sprintf(":%d", port)
	printBanner(addr)
	http.ListenAndServe(addr, nil)
}

func (handler *HttpRouteHandler) handleDashboard(useAuth bool) {
	http.HandleFunc("/", authMiddleware(useAuth, handler.dashboardHandler))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	render(w, "login", nil)
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

	jobs, err := handler.enqueuedJobRepository.FetchDueJobs(ctx, pageNumber, PageSize, statuses, nil)
	if err != nil {
		log.Printf("failed to fetch jobs: %v", err)
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	allJobsCount, _ := handler.enqueuedJobRepository.CountAllJobsGroupedByStatus(ctx)

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
		if err := handler.enqueuedJobRepository.RemoveByID(ctx, jobID); err != nil {
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

func (handler *HttpRouteHandler) handleLogin() {
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			render(w, "login", map[string]interface{}{"HideHeader": true})
		case http.MethodPost:
			username := r.FormValue("username")
			password := r.FormValue("password")

			user, err := handler.userRepository.Find(r.Context(), username, password)
			if err != nil {
				log.Println(err.Error())
				http.SetCookie(w, &http.Cookie{
					Name:     "warning",
					Value:    url.QueryEscape("invalid username or password"),
					Path:     "/",
					MaxAge:   5,
					HttpOnly: false,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			if user != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "auth",
					Value:    generateAuthToken(username),
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "warning",
				Value:    url.QueryEscape("invalid username or password"),
				Path:     "/",
				MaxAge:   5,
				HttpOnly: false,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
}

func (handler HttpRouteHandler) handleLogout() {
	http.HandleFunc("/Logout", func(writer http.ResponseWriter, request *http.Request) {
		http.SetCookie(writer, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		http.Redirect(writer, request, "/login", http.StatusPermanentRedirect)
	})
}
