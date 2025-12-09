package main

import (
	"net/http"
	"log"
	"sync/atomic"
	"fmt"
	"encoding/json"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func main () {
	var apiCfg apiConfig
	apiCfg.fileserverHits.Store(0)

	sMux := http.NewServeMux()
	s := http.Server{}
	s.Addr = ":8080"
	s.Handler = sMux
	fileServer := http.FileServer(http.Dir("."))
	
	sMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))
	sMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK\n"))
	})

	sMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	sMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	sMux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {

		type parameters struct {
			Body string `json:"body"`
		}
		type errorResponse struct {
			ErrorMessage string `json:"error"`
		}
		type validResponse struct {
			Valid bool `json:"valid"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		respError := errorResponse{}

		if err != nil {
			respError.ErrorMessage = "Something went wrong"
			dat, err := json.Marshal(respError)
			if err != nil {
				log.Printf("Error marshalling JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			w.Write(dat)
			return
			}

		if len(params.Body) > 140 {
			respError.ErrorMessage = "Chirp is too long"
			dat, err := json.Marshal(respError)
			if err != nil {
				log.Printf("Error marshalling JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			w.Write(dat)
			return
		}

		dat, err := json.Marshal(validResponse{Valid: true})
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(dat)
	})

	err := s.ListenAndServe()
	if err != nil {
		log.Fatal("fatal error:", err)
	}
}