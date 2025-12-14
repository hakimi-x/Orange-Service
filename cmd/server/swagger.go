//go:build dev

package main

import (
	"log"
	"net/http"

	_ "update-server/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func registerSwagger() {
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	log.Println("Swagger UI 已启用: /swagger/index.html")
}
