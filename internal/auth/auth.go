package auth

import(
	"time"
	"github.com/google/uuid"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil{
		return "", err
	}
	return hashedPassword, nil
}

func ChekPasswordHash(password, hash string) (bool, error) {
	value, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return value, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{}
	claims.Issuer = "chirpy"
	claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().UTC().Add(expiresIn))
	claims.Subject = userID.String()
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	result, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return result, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, 
		func(token *jwt.Token) (interface{}, error) {return []byte(tokenSecret), nil})

	if err != nil {
		return uuid.UUID{}, err
	}
	idStr, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil	
}