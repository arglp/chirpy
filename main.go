package main

import (
	"net/http"
	"log"
	"sync/atomic"
	"fmt"
	"encoding/json"
	"strings"
	"os"
	"database/sql"
	"github.com/arglp/chirpy/internal/database"
	
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		ErrorMessage string `json:"error"`
	}
	respError := errorResponse{
		ErrorMessage: msg,
	}
	respondWithJson(w, code, respError)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func replaceProfaneWords(text string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	replacer := "****"
	words := strings.Split(text, " ")
	for i, word := range words {
		for _, profaneWord := range profaneWords{
			if strings.ToLower(word) == profaneWord{
				words[i] = replacer
			}
		}
	}
	return strings.Join(words, " ")
}

func main () {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("fatal error: ", err)
	}

	var apiCfg apiConfig
	apiCfg.fileserverHits.Store(0)
	apiCfg.dbQueries = database.New(db)

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
		type cleanedBody struct {
			CleanedBody string `json:"cleaned_body"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			respondWithError(w, 400, "Something went wrong")
			return
		}
		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}
		respondWithJson(w, 200, cleanedBody{CleanedBody: replaceProfaneWords(params.Body),})
	})

	err = s.ListenAndServe()
	if err != nil {
		log.Fatal("fatal error:", err)
	}
}