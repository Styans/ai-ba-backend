package main

import (
	"log"
	"os"

	"ai-ba/internal/app"
)

func main() {
	log.Println("starting ai-ba REST API server")
	if err := app.Run(); err != nil {
		log.Println("failed to run app:", err)
		os.Exit(1)
	}
}
