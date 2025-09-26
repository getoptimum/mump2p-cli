package main

import (
	"log"
)

func main() {
	if err := RunE2ETests(); err != nil {
		log.Fatalf("[e2e] failed: %v", err)
	}
	log.Println("[e2e] all tests passed ✔️")
}
