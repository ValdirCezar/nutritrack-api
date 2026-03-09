package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// contextKey tipo customizado para chaves de contexto (evita colisões)
type contextKey string

const (
	// userIDKey chave utilizada para armazenar o ID do usuário no contexto da requisição
	userIDKey contextKey = "userID"
)

// AuthMiddleware retorna um middleware que valida o token JWT em cada requisição protegida.
// Extrai o token do header Authorization (formato: "Bearer <token>"),
// valida a assinatura e expiração, e insere o userID no contexto.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Adicionar rate limiting por IP/usuário neste middleware

			// Extrai o header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "token de autenticação não fornecido")
				return
			}

			// Verifica o formato "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "formato de token inválido. Use: Bearer <token>")
				return
			}

			tokenString := parts[1]

			// Faz o parse e validação do token JWT
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Verifica se o método de assinatura é HMAC (previne ataques de troca de algoritmo)
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("método de assinatura inválido")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				writeError(w, http.StatusUnauthorized, "token inválido ou expirado")
				return
			}

			// Extrai as claims do token
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				writeError(w, http.StatusUnauthorized, "token inválido")
				return
			}

			// Extrai o ID do usuário da claim "sub"
			userIDStr, ok := claims["sub"].(string)
			if !ok || userIDStr == "" {
				writeError(w, http.StatusUnauthorized, "token não contém identificação do usuário")
				return
			}

			// Converte o ID string para ObjectID do MongoDB
			userID, err := primitive.ObjectIDFromHex(userIDStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "identificação do usuário inválida no token")
				return
			}

			// Insere o userID no contexto da requisição para uso nos handlers
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extrai o ID do usuário do contexto da requisição.
// Deve ser chamado apenas em rotas protegidas pelo AuthMiddleware.
func GetUserID(r *http.Request) (primitive.ObjectID, error) {
	userID, ok := r.Context().Value(userIDKey).(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("usuário não autenticado")
	}
	return userID, nil
}

// writeError envia uma resposta de erro em formato JSON
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
