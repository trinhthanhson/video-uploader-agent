package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/internal/uploads/complete", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Println("UPLOAD COMPLETE:", string(body))
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/internal/uploads/failed", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Println("UPLOAD FAILED:", string(body))
		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("Mock server running at http://127.0.0.1:8080")
	_ = http.ListenAndServe(":8080", nil)
}
