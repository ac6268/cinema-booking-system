package main

import (
	"log"
	"net/http"

	"github.com/aman-chandra314/cinema-booking-system/configs"
	"github.com/aman-chandra314/cinema-booking-system/internal/db"
)

func main() {
	cfg := configs.Load()

	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer database.Close()

	log.Printf("server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
