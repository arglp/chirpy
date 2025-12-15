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
	"context"
	"github.com/arglp/chirpy/internal/database"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string	

}
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body		string `json:"body"`
	UserID	uuid.UUID `json:"user_id"`
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
	if cfg.platform != "dev" {
		respondWithError(w, 403, "This action is forbidden")
		return
	}
	err := cfg.dbQueries.DeleteUsers(context.Background())
	if err != nil {
		respondWithError (w, 403, "Couldn't dele users")
		return
	}

	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email	 string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	user, err := cfg.dbQueries.CreateUser(context.Background(), params.Email)
	if err != nil {
		respondWithError(w, 400, "Couldn*t create user")
	}
	respondWithJson(w, 201, User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email})
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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

	chirp, err := cfg.dbQueries.CreateChirp(context.Background(), database.CreateChirpParams{
		Body: replaceProfaneWords(params.Body),
		UserID: params.UserID})

	if err != nil {
		respondWithError(w, 400, "Coudn't create chirp")
		return
	}
	respondWithJson(w, 201, Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:	chirp.Body,
		UserID: chirp.UserID,
	})	
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
	apiCfg.platform = os.Getenv("PLATFORM")

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

	sMux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)



	err = s.ListenAndServe()
	if err != nil {
		log.Fatal("fatal error:", err)
	}
}