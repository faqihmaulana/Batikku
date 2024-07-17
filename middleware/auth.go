package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}


func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionToken, err := ctx.Cookie("session_token")
		if err != nil {
			log.Printf("No session token found: %v", err)
			if ctx.GetHeader("Content-Type") == "application/json" {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized, please login"})
			} else {
				ctx.Redirect(http.StatusSeeOther, "/login")
				ctx.Abort()
			}
			return
		}

		tokenClaims := &Claims{}
		token, err := jwt.ParseWithClaims(sessionToken, tokenClaims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret-key"), nil // Ganti "secret-key" dengan secret key yang sesuai
		})

		if err != nil || !token.Valid {
			log.Printf("Token parsing error: %v", err)
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Malformed token"})
					return
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token is expired or not valid yet"})
					return
				}
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		ctx.Set("email", tokenClaims.Email)
		log.Printf("Token valid for user: %s", tokenClaims.Email)

		ctx.Next()
	}
}