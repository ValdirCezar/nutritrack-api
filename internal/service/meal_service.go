package service

import (
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MealService contém a lógica de negócio para registro e consulta de refeições
type MealService struct {
	mealRepo         *repository.MealRepository
	openaiService    *OpenAIService
	foodCacheService *FoodCacheService
}

// NewMealService cria uma nova instância do serviço de refeições
func NewMealService(
	mealRepo *repository.MealRepository,
	openaiService *OpenAIService,
	foodCacheService *FoodCacheService,
) *MealService {
	return &MealService{
		mealRepo:         mealRepo,
		openaiService:    openaiService,
		foodCacheService: foodCacheService,
	}
}

// RegisterMeal registra uma nova refeição para o usuário.
// Analisa a descrição dos alimentos via OpenAI e salva os dados nutricionais.
func (s *MealService) RegisterMeal(userID primitive.ObjectID, req model.MealRequest) (*model.Meal, error) {
	// Validação da descrição
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return nil, errors.New("descrição da refeição não pode ser vazia")
	}
	if len(description) > 500 {
		return nil, errors.New("descrição da refeição deve ter no máximo 500 caracteres")
	}

	// Chama a OpenAI para analisar os alimentos descritos
	foodResponse, err := s.openaiService.AnalyzeFood(description)
	if err != nil {
		return nil, err
	}

	// Salva cada alimento no cache para uso futuro (não bloqueia em caso de erro)
	for _, food := range foodResponse.Foods {
		if err := s.foodCacheService.SaveToCache(food); err != nil {
			log.Printf("Aviso: erro ao salvar alimento '%s' no cache: %v", food.Name, err)
		}
	}

	// Arredonda os totais para 1 casa decimal
	totals := model.NutrientTotals{
		Protein:  math.Round(foodResponse.Totals.Protein*10) / 10,
		Carbs:    math.Round(foodResponse.Totals.Carbs*10) / 10,
		Fat:      math.Round(foodResponse.Totals.Fat*10) / 10,
		Calories: math.Round(foodResponse.Totals.Calories*10) / 10,
	}

	// Cria a refeição com a data de hoje no formato YYYY-MM-DD
	meal := &model.Meal{
		UserID:      userID,
		Date:        time.Now().Format("2006-01-02"),
		Description: description,
		Foods:       foodResponse.Foods,
		Totals:      totals,
	}

	// Salva no banco de dados
	if err := s.mealRepo.Create(meal); err != nil {
		return nil, err
	}

	return meal, nil
}

// GetMealsByDate retorna todas as refeições de um usuário em uma data específica
func (s *MealService) GetMealsByDate(userID primitive.ObjectID, date string) ([]model.Meal, error) {
	if date == "" {
		// Se a data não for informada, usa a data de hoje
		date = time.Now().Format("2006-01-02")
	}

	return s.mealRepo.FindByUserIDAndDate(userID, date)
}

// DeleteMeal remove uma refeição, garantindo que pertence ao usuário autenticado
func (s *MealService) DeleteMeal(mealID, userID primitive.ObjectID) error {
	return s.mealRepo.DeleteByID(mealID, userID)
}
