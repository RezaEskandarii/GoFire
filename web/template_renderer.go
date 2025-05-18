package web

import (
	"fmt"
	"html/template"
	"net/http"
)

var (
	baseTmpl = template.Must(template.ParseFiles("web/templates/layout.html"))
)

func render(w http.ResponseWriter, tmplName string, data any) {
	cloned, _ := baseTmpl.Clone()
	fileName := fmt.Sprintf("web/%s.html", tmplName)
	cloned = template.Must(cloned.ParseFiles(fileName))
	cloned.ExecuteTemplate(w, "layout.html", data)
}
