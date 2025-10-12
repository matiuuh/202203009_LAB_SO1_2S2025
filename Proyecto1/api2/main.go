package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Estructura para responder en formato JSON
type Msg struct {
	Mensaje         string `json:"mensaje"`
	API             string `json:"api"`
	VM              string `json:"vm"`
	Estudiante      string `json:"estudiante"`
	Carnet          string `json:"carnet"`
	RespuestaRemota string `json:"respuestaRemota"`
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

// Función para verificar el estado de una API remota
func checkRemoteAPI(url string) string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/healthz")
	if err != nil {
		return fmt.Sprintf("No se pudo contactar la API en %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return fmt.Sprintf("API en %s está funcionando", url)
	}
	return fmt.Sprintf("API en %s respondió con código %d", url, resp.StatusCode)
}

// Handler para llamar a otra API
func callOtherAPI(apiName, vmName, studentName, carnet, targetName, targetURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := Msg{
			Mensaje:    fmt.Sprintf("Hola, responde la API: %s en la %s, desarrollada por el estudiante %s con carnet: %s", apiName, vmName, studentName, carnet),
			API:        apiName,
			VM:         vmName,
			Estudiante: studentName,
			Carnet:     carnet,
		}

		// Aquí verificamos si la API remota está viva
		msg.RespuestaRemota = checkRemoteAPI(targetURL)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(msg)
	}
}

func main() {
	// Credenciales de esta API
	apiName := "API2"
	vmName := "202203009_2"
	studentName := "Mateo Noriega"
	carnet := "202203009"
	port := "8082"

	// URLs de las otras APIs (ajusta según tus VMs)
	api1URL := "http://192.168.122.217:8081"
	api3URL := "http://192.168.122.30:8083"

	mux := http.NewServeMux()

	// Endpoints de esta API
	mux.HandleFunc("/api2/"+carnet+"/llamar-api1", callOtherAPI(apiName, vmName, studentName, carnet, "API1", api1URL))
	mux.HandleFunc("/api2/"+carnet+"/llamar-api3", callOtherAPI(apiName, vmName, studentName, carnet, "API3", api3URL))

	// Endpoint de salud (útil para probar si está vivo)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Endpoint raíz
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("API2 viva. Usa /healthz o /api2/202203009/llamar-api{1|3}\n"))
	})

	// Iniciar servidor con logger
	log.Printf("%s escuchando en puerto %s ...", apiName, port)
	if err := http.ListenAndServe(":"+port, logger(mux)); err != nil {
		log.Fatal(err)
	}
}
