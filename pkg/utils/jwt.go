package utils

import (
    "github.com/golang-jwt/jwt/v5"
	
)

var secretKey = "R1clvDeLZgp5knHvm0WLkBvqMD51khuRBzw1BTjXjH8="

func GenerateToken(role string) (string, error) {
    
    claims := jwt.MapClaims{
        "role":  role,
    }
 
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secretKey)
}