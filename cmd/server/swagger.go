//go:build dev

package main

import (
	"log"
	"net/http"

	_ "update-server/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func registerSwagger(addr string) {
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	log.Printf("Swagger UI: http://%s/swagger/index.html", addr)
}
