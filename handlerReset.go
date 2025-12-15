package main

import(
	"net/http"
	"context"
)

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