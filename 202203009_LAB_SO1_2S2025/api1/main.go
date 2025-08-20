package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Estructura para responder en formato JSON
type Msg struct {
	Mensaje    string `json:"mensaje"`
	API        string `json:"api"`
	VM         string `json:"vm"`
	Estudiante string `json:"estudiante"`
	Carnet     string `json:"carnet"`
}

// Middleware para registrar logs de cada petición
func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("→ %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("← %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

// Handler que responde en formato JSON
func apiHandler(apiName, vmName, studentName, carnet string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Dividir la URL en partes
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.NotFound(w, r)
			return
		}

		// Construir el mensaje de respuesta
		msg := Msg{
			Mensaje:    fmt.Sprintf("Hola, responde la API: %s en la %s, desarrollada por el estudiante %s con carnet: %s", apiName, vmName, studentName, carnet),
			API:        apiName,
			VM:         vmName,
			Estudiante: studentName,
			Carnet:     carnet,
		}

		// Responder en JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
	}
}

func main() {
	// Credenciales
	apiName := "API1"
	vmName := "202203009_1"
	studentName := "Mateo Noriega"
	carnet := "202203009"
	port := "80"

	mux := http.NewServeMux()

	// Endpoints de esta API
	mux.HandleFunc("/api1/"+carnet+"/llamar-api2", apiHandler(apiName, vmName, studentName, carnet))
	mux.HandleFunc("/api1/"+carnet+"/llamar-api3", apiHandler(apiName, vmName, studentName, carnet))

	// Endpoint de salud (útil para probar si está vivo)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("API1 viva. Usa /healthz o /api1/202203009/llamar-api{2|3}\n"))
	})

	// Iniciar servidor con logger
	log.Printf("%s escuchando en puerto %s ...", apiName, port)
	if err := http.ListenAndServe(":"+port, logger(mux)); err != nil {
		log.Fatal(err)
	}
}
