package auth

import (
	"testing"
	"time"
	"net/http"
	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {

	tests := []struct {
		user			uuid.UUID
		name          	string
		tokenCreate   	string
		tokenValidate 	string
		expiresIn	  	time.Duration
		wantErr       	bool
	}{
		{
			user:			uuid.New(),
			name:      		"correct token not expired",
			tokenCreate:  	"thisisasupersecrettoken",
			tokenValidate:  "thisisasupersecrettoken",
			expiresIn: 		time.Minute,
			wantErr:   		false,
		},
		{
			user:			uuid.New(),
			name:      		"false token not expired",
			tokenCreate:  	"thisisasupersecrettoken",
			tokenValidate:  "thisisnotasupersecrettoken",
			expiresIn: 		time.Minute,
			wantErr:   		true,
		},
		{
			user:			uuid.New(),
			name:      		"correct token expired",
			tokenCreate:  	"thisisasupersecrettoken",
			tokenValidate:  "thisisasupersecrettoken",
			expiresIn: 		-time.Minute,
			wantErr:   		true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokenString, err := MakeJWT(tc.user, tc.tokenCreate, tc.expiresIn)
			if err != nil {
				t.Fatalf("MakeJWT() error = %v", err)
				return
			}
			result, err := ValidateJWT(tokenString, tc.tokenValidate)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("ValidateJWT() error = %v", err)
				}
				return
			}
			if tc.user != result {
				t.Fatalf("ValidateJWT() users don't match")
			}
		})
	}
}





func TestHashPassword(t *testing.T) {
	pass := "swordfish-123!"

	hash1, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash1 == "" {
		t.Fatalf("HashPassword() returned empty hash")
	}

	hash2, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword() error on second call = %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("expected different hashes for same password, got equal")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name          string
		password      string
		hash          string
		wantErr       bool
		wantMatch     bool
	}{
		{
			name:      "correct password",
			password:  password,
			hash:      hash,
			wantErr:   false,
			wantMatch: true,
		},
		{
			name:      "wrong password",
			password:  "wrong-password",
			hash:      hash,
			wantErr:   false,
			wantMatch: false,
		},
		{
			name:      "invalid hash",
			password:  password,
			hash:      "not-a-real-hash",
			wantErr:   true,
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tc.password, tc.hash)

			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && match != tc.wantMatch {
				t.Fatalf("match = %v, wantMatch = %v", match, tc.wantMatch)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	header := http.Header{}
	expectedToken := "thisisthetoken"
	_, err := GetBearerToken(header)
	if err == nil {
		t.Fatalf("GetBearerToken: expected error, no Authorization header")
		return
	}
	header.Add("Authorization", "Bearer thisisthetoken")
	token, err := GetBearerToken(header)
	if err != nil {
		t.Fatalf("GetBearerToken() error: %v", err)
		return
	}
	if expectedToken != token {
		t.Fatalf("Token not correct")
	}
}