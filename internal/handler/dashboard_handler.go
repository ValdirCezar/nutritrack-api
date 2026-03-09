package handler

import (
	"math"
	"net/http"
	"time"

	"github.com/valdircezar/nutritrack-api/internal/middleware"
	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/service"
)

// DashboardHandler gerencia as requisições HTTP do dashboard nutricional diário
type DashboardHandler struct {
	mealService    *service.MealService
	profileService *service.ProfileService
}

// NewDashboardHandler cria uma nova instância do handler do dashboard
func NewDashboardHandler(mealService *service.MealService, profileService *service.ProfileService) *DashboardHandler {
	return &DashboardHandler{
		mealService:    mealService,
		profileService: profileService,
	}
}

// Get processa GET /api/dashboard?date=YYYY-MM-DD — retorna o resumo diário.
// Consolida metas do perfil, consumo realizado e cálculos de progresso.
// Requer autenticação JWT e perfil configurado.
func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Extrai o userID do contexto JWT
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Usuário não autenticado")
		return
	}

	// Busca o perfil do usuário (necessário para as metas)
	profile, err := h.profileService.GetProfile(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar perfil")
		return
	}

	if profile == nil {
		respondWithError(w, http.StatusNotFound, "Perfil não encontrado. Complete o onboarding primeiro.")
		return
	}

	// Obtém a data do query parameter (padrão: hoje)
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Busca todas as refeições do dia
	meals, err := h.mealService.GetMealsByDate(userID, date)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Erro ao buscar refeições")
		return
	}

	// Calcula os totais consumidos somando todas as refeições do dia
	consumed := model.NutrientTotals{}
	for _, meal := range meals {
		consumed.Protein += meal.Totals.Protein
		consumed.Carbs += meal.Totals.Carbs
		consumed.Fat += meal.Totals.Fat
		consumed.Calories += meal.Totals.Calories
	}

	// Arredonda totais consumidos para 1 casa decimal
	consumed.Protein = math.Round(consumed.Protein*10) / 10
	consumed.Carbs = math.Round(consumed.Carbs*10) / 10
	consumed.Fat = math.Round(consumed.Fat*10) / 10
	consumed.Calories = math.Round(consumed.Calories*10) / 10

	// Metas diárias do perfil
	goals := model.NutrientTotals{
		Protein:  profile.DailyProtein,
		Carbs:    profile.DailyCarbs,
		Fat:      profile.DailyFat,
		Calories: profile.DailyCalories,
	}

	// Calcula o restante (pode ser negativo se ultrapassou a meta)
	remaining := model.NutrientTotals{
		Protein:  math.Round((goals.Protein-consumed.Protein)*10) / 10,
		Carbs:    math.Round((goals.Carbs-consumed.Carbs)*10) / 10,
		Fat:      math.Round((goals.Fat-consumed.Fat)*10) / 10,
		Calories: math.Round((goals.Calories-consumed.Calories)*10) / 10,
	}

	// Calcula porcentagem de consumo em relação às metas
	percentage := model.NutrientTotals{
		Protein:  calculatePercentage(consumed.Protein, goals.Protein),
		Carbs:    calculatePercentage(consumed.Carbs, goals.Carbs),
		Fat:      calculatePercentage(consumed.Fat, goals.Fat),
		Calories: calculatePercentage(consumed.Calories, goals.Calories),
	}

	// Monta a resposta do dashboard
	dashboard := model.DashboardResponse{
		Date:       date,
		Goals:      goals,
		Consumed:   consumed,
		Remaining:  remaining,
		Percentage: percentage,
		Meals:      meals,
	}

	respondWithJSON(w, http.StatusOK, dashboard)
}

// calculatePercentage calcula a porcentagem de consumo em relação à meta.
// Retorna 0 se a meta for zero para evitar divisão por zero.
func calculatePercentage(consumed, goal float64) float64 {
	if goal == 0 {
		return 0
	}
	return math.Round((consumed/goal)*1000) / 10
}
