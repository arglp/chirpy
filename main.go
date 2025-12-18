package main

import (
	"net/http"
	"log"
	"os"
	"database/sql"
	"github.com/arglp/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

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
	apiCfg.platform = os.Getenv("PLATFORM")
	apiCfg.secret = os.Getenv("SECRET")

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
	sMux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
	sMux.HandleFunc("POST /api/chirps", apiCfg.handlerPostChirps)
	sMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	sMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)
	sMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	sMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	sMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	err = s.ListenAndServe()
	if err != nil {
		log.Fatal("fatal error:", err)
	}
}