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
		http.Error(w, "Invalid zipcode format", http.StatusUnprocessableEntity)
		return
	}

	resp, statusCode, err := fetchWeather(req.CEP)
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func fetchWeather(cep string) ([]byte, int, error) {
	resp, err := http.Get("http://service-b:8081/weather?cep=" + cep)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to communicate with service B: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to read response from service B: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("%s", string(body))
	}

	return body, http.StatusOK, nil
}

func main() {
	http.Handle("/zipcode", http.HandlerFunc(handleZipcodeRequest))
	fmt.Println("Service A running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
