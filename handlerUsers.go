package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/arglp/chirpy/internal/auth"
	"github.com/arglp/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token 	  string	`json:"token"`
}

func transcribeUser(dU database.User) User {
	return User{
		ID:	dU.ID,
		CreatedAt: dU.CreatedAt,
		UpdatedAt: dU.UpdatedAt,
		Email: dU.Email,
	}
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email	 string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	} 

	user, err := cfg.dbQueries.CreateUser(context.Background(), database.CreateUserParams{
		Email: params.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		respondWithError(w, 400, "Couldn*t create user")
	}
	respondWithJson(w, 201, transcribeUser(user))
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email	 string `json:"email"`
		ExpiresInSeconds int64 `json:"expires_in_seconds"`
	}


	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}

	expiresIn := time.Duration(params.ExpiresInSeconds) * time.Second
	if expiresIn > time.Hour {
		expiresIn = time.Hour
	}
	if expiresIn == 0 {
		expiresIn = time.Hour
	}

	user, err := cfg.dbQueries.GetUser(context.Background(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "Couldn't check password")
		return
	}
	if !ok {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	tokenString, err := auth.MakeJWT(user.ID, cfg.secret, time.Duration(expiresIn))
	if err != nil {
		respondWithError(w, 401, "Couldn't make JWT")
	}
	
	jsonUser := transcribeUser(user)
	jsonUser.Token = tokenString


	respondWithJson(w, 200, jsonUser)
}