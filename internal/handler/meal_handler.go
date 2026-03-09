package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/valdircezar/nutritrack-api/internal/middleware"
	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MealHandler gerencia as requisições HTTP relacionadas às refeições
type MealHandler struct {
	mealService *service.MealService
}

// NewMealHandler cria uma nova instância do handler de refeições
func NewMealHandler(mealService *service.MealService) *MealHandler {
	return &MealHandler{
		mealService: mealService,
	}
}

// Register processa POST /api/meals — registra uma nova refeição.
// Requer autenticação JWT. Recebe a descrição dos alimentos e retorna a refeição
// com os dados nutricionais calculados pela OpenAI.
func (h *MealHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Decodifica o corpo da requisição
	var req model.MealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Dados inválidos no corpo da requisição")
		return
	}

	// Chama o serviço para registrar a refeição
	meal, err := h.mealService.RegisterMeal(userID, req)
	if err != nil {
		// Erros de validação conhecidos são seguros para retornar ao cliente
		switch err.Error() {
		case "descrição da refeição não pode ser vazia",
			"descrição da refeição deve ter no máximo 500 caracteres",
			"descrição do alimento não pode ser vazia",
			"a OpenAI não identificou nenhum alimento na descrição",
			"erro ao analisar alimentos. Tente novamente em alguns instantes":
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			// Erros internos não devem vazar detalhes para o cliente
			log.Printf("Erro ao registrar refeição: %v", err)
			respondWithError(w, http.StatusInternalServerError, "erro ao processar a refeição. Tente novamente")
		}
		return
	}

	respondWithJSON(w, http.StatusCreated, meal)
}

// ListByDate processa GET /api/meals?date=YYYY-MM-DD — lista refeições do dia.
// Requer autenticação JWT. Se a data não for informada, usa a data atual.
func (h *MealHandler) ListByDate(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Obtém a data do query parameter (opcional)
	date := r.URL.Query().Get("date")

	// Busca as refeições do dia
	meals, err := h.mealService.GetMealsByDate(userID, date)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar refeições")
		return
	}

	respondWithJSON(w, http.StatusOK, meals)
}

// Delete processa DELETE /api/meals/{id} — remove uma refeição pelo ID.
// Requer autenticação JWT. Apenas o dono da refeição pode removê-la.
func (h *MealHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Extrai o ID da refeição do path
	// Espera o formato: /api/meals/{id}
	// O roteamento deve passar o ID no path
	mealIDStr := r.PathValue("id")
	if mealIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "ID da refeição é obrigatório")
		return
	}

	mealID, err := primitive.ObjectIDFromHex(mealIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "ID da refeição inválido")
		return
	}

	// Chama o serviço para deletar a refeição
	if err := h.mealService.DeleteMeal(mealID, userID); err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Refeição removida com sucesso",
	})
}
