package web

import (
	"fmt"
	"github.com/RezaEskandarii/gofire/pgk/models"
	"net/http"
	"strconv"
)

type DataMap struct {
	Data map[string]interface{}
}

func NewPaginatedDataMap[T any](data models.PaginationResult[T]) DataMap {
	return DataMap{
		Data: map[string]interface{}{
			"Page":            data.Page,
			"TotalPages":      data.TotalPages,
			"Items":           data.Items,
			"HasPreviousPage": data.HasPreviousPage,
			"HasNextPage":     data.HasNextPage,
			"TotalItems":      data.TotalItems,
		},
	}
}

func (d DataMap) Add(key string, value interface{}) DataMap {
	d.Data[key] = value
	return d
}

func getPageNumber(r *http.Request) int {
	page := r.URL.Query().Get("page")
	pageNumber, err := strconv.ParseInt(page, 10, 64)
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}
	return int(pageNumber)
}

func printBanner(port string) {
	host := "localhost"
	scheme := "http"
	appAddr := fmt.Sprintf("%s://%s%s", scheme, host, port)
	width := 60
	fmt.Println("############################################################")
	fmt.Printf("# %-*s #\n", width-4, "")
	fmt.Printf("# %-*s #\n", width-4, "GoFire Started")
	fmt.Printf("# %-*s%s #\n", width-25, "GoFire Server is starting at ", appAddr)
	fmt.Printf("# %-*s #\n", width-4, "")
	fmt.Println("############################################################")
}
