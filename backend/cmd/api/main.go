package main

import (
	"log"
	"net/http"
	"os"

	"order-stock/backend/internal/api"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8091"
	}
	log.Printf("API listening on %s", address)
	if err := http.ListenAndServe(address, api.NewServer()); err != nil {
		log.Fatal(err)
	}
}
