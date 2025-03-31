package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelzipkin "go.opentelemetry.io/otel/exporters/zipkin" // Alias para evitar conflito
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace" // Alias para o SDK trace
	"go.opentelemetry.io/otel/trace"              // Interface trace
)

var tracer trace.Tracer

type TemperatureResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

// initTracer inicializa o tracer do OpenTelemetry com o exportador Zipkin.
func initTracer() trace.Tracer {
	exporter, err := otelzipkin.New("http://localhost:9411/api/v2/spans") // Usando a função correta
	if err != nil {
		log.Fatalf("failed to create Zipkin exporter: %v", err)
	}

	tp := sdktrace.NewTracerProvider( // Usando o alias sdktrace
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Tracer("service-b")
}

// WeatherResponse define a estrutura da resposta JSON para o endpoint de clima.
type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

// fetchCityFromCEP busca a cidade correspondente ao CEP usando a API ViaCEP.
func fetchCityFromCEP(cep string) (string, error) {
	resp, err := http.Get("https://viacep.com.br/ws/" + cep + "/json/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CEP not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	city := result["localidade"].(string)
	return city, nil
}

// fetchTemperature busca a temperatura atual de uma cidade usando a API WeatherAPI.
func fetchTemperature(city string) (WeatherResponse, error) {
	// Monta a URL da API WeatherAPI
	encodedCity := url.QueryEscape(city)
	url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", "b2d1b00af1124f7fb2c173842251802", encodedCity)

	// Faz a requisição HTTP
	resp, err := http.Get(url)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("failed to fetch temperature: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("failed to fetch temperature, status: %d", resp.StatusCode)
	}

	// Lê o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("failed to read response body: %v", err)
	}

	// Decodifica a resposta JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return WeatherResponse{}, fmt.Errorf("failed to decode JSON: %v", err)
	}

	// Extrai a temperatura em Celsius
	current, ok := result["current"].(map[string]interface{})
	if !ok {
		return WeatherResponse{}, fmt.Errorf("invalid response format")
	}

	tempC, ok := current["temp_c"].(float64)
	if !ok {
		return WeatherResponse{}, fmt.Errorf("invalid temperature format")
	}

	// Converte para Fahrenheit e Kelvin
	tempF := tempC*1.8 + 32
	tempK := tempC + 273

	return WeatherResponse{
		City:  city,
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}, nil
}

// handleWeatherRequest lida com as requisições para o endpoint de clima.
func handleWeatherRequest(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleWeatherRequest")
	defer span.End()

	cep := r.URL.Query().Get("cep")
	if len(cep) != 8 {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	city, err := fetchCityFromCEP(cep)
	if err != nil {
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	response, _ := fetchTemperature(city)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Inicializa o tracer.
	tracer = initTracer()

	// Configura o endpoint HTTP.
	http.Handle("/weather", otelhttp.NewHandler(http.HandlerFunc(handleWeatherRequest), "handleWeatherRequest"))
	fmt.Println("Service B running on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
