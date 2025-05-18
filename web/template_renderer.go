package web

import (
	"fmt"
	"gofire/internal/state"
	"html/template"
	"net/http"
)

func render(w http.ResponseWriter, tmplName string, data any) {
	funcMap := template.FuncMap{
		"StatusBadgeClass": StatusBadgeClass,
		"StatusBgClass":    StatusBgClass,
	}

	tmpl := template.New("layout.html").Funcs(funcMap)

	tmpl = template.Must(tmpl.ParseFiles(
		"web/templates/layout.html",
		fmt.Sprintf("web/templates/%s.html", tmplName),
	))

	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StatusBadgeClass(status state.JobStatus) string {
	switch status {
	case state.StatusQueued:
		return "badge bg-info"
	case state.StatusProcessing:
		return "badge bg-primary"
	case state.StatusSucceeded:
		return "badge bg-success"
	case state.StatusFailed:
		return "badge bg-danger"
	case state.StatusRetrying:
		return "badge bg-warning"
	case state.StatusCancelled:
		return "badge bg-secondary"
	case state.StatusExpired:
		return "badge bg-warning"
	case state.StatusDead:
		return "badge bg-dark"
	default:
		return "badge bg-light text-dark"
	}
}

func StatusBgClass(status state.JobStatus) string {
	switch status {
	case state.StatusQueued:
		return " bg-info"
	case state.StatusProcessing:
		return " bg-primary"
	case state.StatusSucceeded:
		return " bg-success"
	case state.StatusFailed:
		return " bg-danger"
	case state.StatusRetrying:
		return " bg-warning"
	case state.StatusCancelled:
		return " bg-secondary"
	case state.StatusExpired:
		return " bg-warning"
	case state.StatusDead:
		return " bg-dark"
	default:
		return " bg-light text-dark"
	}
}
