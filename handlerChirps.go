package main

import(
	"time"
	"encoding/json"
	"net/http"
	"context"

	"github.com/arglp/chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body		string `json:"body"`
	UserID	uuid.UUID `json:"user_id"`
}

func transcribeChirp(dC database.Chirp) Chirp {
	return Chirp{
		ID:	dC.ID,
		CreatedAt: dC.CreatedAt,
		UpdatedAt: dC.UpdatedAt,
		Body: dC.Body,
		UserID: dC.UserID,
	}
}



func (cfg *apiConfig) handlerPostChirps(w http.ResponseWriter, r *http.Request) {
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
	respondWithJson(w, 201, transcribeChirp(chirp))	
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	
	results, err := cfg.dbQueries.GetChirps(context.Background())
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	var chirps []Chirp

	for _, result := range results {
		chirps = append(chirps, transcribeChirp(result))
	}
	respondWithJson(w, 200, chirps)	
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
	}
	
	chirp, err := cfg.dbQueries.GetChirp(context.Background(), chirpID)
	if err != nil {
		respondWithError(w, 404, "couldn't find chirp")
		return
	}

	respondWithJson(w, 200, transcribeChirp(chirp))	
}