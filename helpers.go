package main

import(
	"net/http"
	"encoding/json"
	"log"
	"strings"
)

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