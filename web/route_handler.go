package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"gofire/internal/repository"
	"gofire/internal/state"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

const (
	PageSize = 15
)

type HttpRouteHandler struct {
	enqueuedJobRepository repository.EnqueuedJobRepository
	cronJobRepository     repository.CronJobRepository
	userRepository        repository.UserRepository
	SecretKey             string
	UseAuth               bool
	Port                  int
}

func NewRouteHandler(
	repository repository.EnqueuedJobRepository,
	userRepository repository.UserRepository,
	cronJobRepository repository.CronJobRepository,
	secretKey string,
	useAuth bool,
	port int,
) HttpRouteHandler {
	return HttpRouteHandler{
		enqueuedJobRepository: repository,
		cronJobRepository:     cronJobRepository,
		userRepository:        userRepository,
		SecretKey:             secretKey,
		UseAuth:               useAuth,
		Port:                  port,
	}
}

func (handler *HttpRouteHandler) Serve() {
	handler.handleEnqueued()
	handler.handleCronJobs()
	handler.handleChangeCronJobStatus()
	handler.handleCharts()
	handler.handleIndex()
	handler.handleLogin()
	handler.handleLogout()
	addr := fmt.Sprintf(":%d", handler.Port)
	printBanner(addr)
	// Serve embedded static files
	staticContent, _ := fs.Sub(staticFiles, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))
	http.ListenAndServe(addr, nil)
}

func (handler *HttpRouteHandler) handleEnqueued() {
	http.HandleFunc("/enqueued", handler.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
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

		render(w, "enqueued", data.Data)
	}))
}

func (handler *HttpRouteHandler) handleIndex() {
	http.HandleFunc("/", handler.authMiddleware(func(writer http.ResponseWriter, request *http.Request) {
		if isAuthenticated(request, handler.SecretKey) {
			http.Redirect(writer, request, "charts", http.StatusSeeOther)
			return
		}
		http.Redirect(writer, request, "login", http.StatusSeeOther)
	}))
}

func (handler *HttpRouteHandler) handleCharts() {
	http.HandleFunc("/charts", handler.authMiddleware(func(writer http.ResponseWriter, request *http.Request) {

		loadCharts := request.URL.Query().Get("loadCharts")
		if loadCharts != "" {
			enqueuedJobs, err := handler.enqueuedJobRepository.CountAllJobsGroupedByStatus(request.Context())
			if err != nil {
				http.Error(writer, "Failed to get enqueued jobs", http.StatusInternalServerError)
				log.Println("Error in enqueuedJobs:", err)
				return
			}

			cronJobs, err := handler.cronJobRepository.CountAllJobsGroupedByStatus(request.Context())
			if err != nil {
				http.Error(writer, "Failed to get cron jobs", http.StatusInternalServerError)
				log.Println("Error in cronJobs:", err)
				return
			}

			data := map[string]interface{}{
				"enqueued_jobs": enqueuedJobs,
				"cron_jobs":     cronJobs,
			}

			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusOK)

			if err := json.NewEncoder(writer).Encode(data); err != nil {
				log.Println("Error encoding JSON:", err)
			}
			return
		}

		render(writer, "charts", nil)
	}))
}

func (handler *HttpRouteHandler) handleCronJobs() {
	http.HandleFunc("/cron-jobs", handler.authMiddleware(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		pageNumber := getPageNumber(r)
		statusParam := strings.TrimSpace(r.URL.Query().Get("status"))
		status := state.JobStatus(statusParam)

		jobs, err := handler.cronJobRepository.GetAll(ctx, pageNumber, PageSize, status)
		if err != nil {
			log.Printf("failed to fetch jobs: %v", err)
			http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
			return
		}

		allJobsCount, _ := handler.cronJobRepository.CountAllJobsGroupedByStatus(ctx)

		data := NewPaginatedDataMap(*jobs).
			Add("Statuses", state.AllStatuses).
			Add("CurrentStatus", status).
			Add("JobsCount", allJobsCount)

		render(w, "cron", data.Data)

	}))
}

func (handler *HttpRouteHandler) handleChangeCronJobStatus() {
	http.HandleFunc("/change-cron-job-status", handler.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		action := r.FormValue("action")
		idStr := r.FormValue("id")

		if idStr == "" || (action != "activate" && action != "deactivate") {
			http.Error(w, "Invalid Parameters", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		log.Printf("Changing status of job ID %s to %s\n", idStr, action)

		message := ""
		switch action {
		case "activate":
			handler.cronJobRepository.Activate(r.Context(), id)
			message = "Job activated successfully!"
			break
		case "deactivate":
			handler.cronJobRepository.DeActivate(r.Context(), id)
			message = "Job deactivated successfully!"
			break
		default:
			http.Error(w, "Invalid Action", http.StatusBadRequest)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "flash",
			Value:    url.QueryEscape(message),
			Path:     "/",
			MaxAge:   5,
			HttpOnly: false,
		})

		http.Redirect(w, r, "/cron-jobs", http.StatusSeeOther)
	}))
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
	http.Redirect(w, r, "/enqueued", http.StatusSeeOther)

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
					Value:    generateAuthToken(username, handler.SecretKey),
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/charts", http.StatusSeeOther)
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

func (handler *HttpRouteHandler) handleLogout() {

	http.HandleFunc("/Logout", func(writer http.ResponseWriter, request *http.Request) {
		http.SetCookie(writer, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		http.Redirect(writer, request, "/login", http.StatusSeeOther)
	})
}
