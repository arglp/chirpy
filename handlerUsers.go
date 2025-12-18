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
	IsChirpyRed bool	`json:"is_chirpy_red"`
	Token 	  string	`json:"token"`
	RefreshToken	string`json:"refresh_token"`
}

func transcribeUser(dU database.User) User {
	return User{
		ID:	dU.ID,
		CreatedAt: dU.CreatedAt,
		UpdatedAt: dU.UpdatedAt,
		Email: dU.Email,
		IsChirpyRed: dU.IsChirpyRed,
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
	}


	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}

	expiresIn := time.Hour

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

	refreshTokenString, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 401, "Couldn't make refresh token")
	}

	refreshToken, err := cfg.dbQueries.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		Token: refreshTokenString,
		UserID: user.ID,
		ExpiresAt: time.Now().AddDate(0, 0, 60),
	})

	if err != nil {
		respondWithError(w, 400, "something went wrong")
	}
	
	jsonUser := transcribeUser(user)
	jsonUser.Token = tokenString
	jsonUser.RefreshToken = refreshToken.Token

	respondWithJson(w, 200, jsonUser)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		return
	}

	refreshToken, err := cfg.dbQueries.GetUserFromRefreshToken(context.Background(), token)
	if err != nil {
		respondWithError(w, 401, "couldn't find refresh token")
		return
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		respondWithError(w, 401, "refresh token expired")
		return
	}
	if refreshToken.RevokedAt.Valid {
		respondWithError(w, 401, "refresh token revoked")
		return
	}

	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, 401, "something went wrong")
		return
	}

	respondWithJson(w, 200, response{Token: accessToken})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "something went wrong")
		return
	}
	err = cfg.dbQueries.RevokeRefreshToken(context.Background(), token)
	if err != nil {
		respondWithError(w, 401, "couldn't find refresh token")
		return
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "couldn't find access code")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 401, "couldn't hash password")
		return
	}

	id, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, 401, "couldn't vaildate access code")
		return
	}

	user, err := cfg.dbQueries.SetUserEmailPassword(context.Background(), database.SetUserEmailPasswordParams{
		Email: params.Email,
		HashedPassword: hashedPassword,
		ID: id,
	})

	if err != nil {
		respondWithError(w, 401, "couldn't get user")
		return
	}

	jsonUser := transcribeUser(user)

	respondWithJson(w, 200, jsonUser)
}