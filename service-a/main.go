package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type ZipcodeRequest struct {
	CEP string `json:"cep"`
}

func validateZipcode(cep string) bool {
	if len(cep) != 8 {
		return false
	}
	if _, err := strconv.Atoi(cep); err != nil {
		return false
	}
	return true
}

func handleZipcodeRequest(w http.ResponseWriter, r *http.Request) {

	var req ZipcodeRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil || !validateZipcode(req.CEP) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	resp, err := fetchWeather(req.CEP)
	if err != nil {
		http.Error(w, "error communicating with service B", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func fetchWeather(cep string) ([]byte, error) {
	resp, err := http.Get("http://localhost:8081/weather?cep=" + cep)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch temperature, status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func main() {
	http.Handle("/zipcode", http.HandlerFunc(handleZipcodeRequest))
	fmt.Println("Service A running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
