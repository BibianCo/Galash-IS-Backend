package main

import (
	"log"

	"github.com/galash-uptc/backend/cmd/api"
)

func main() {
	if err := api.Run(); err != nil {
		log.Fatal(err)
	}
}
