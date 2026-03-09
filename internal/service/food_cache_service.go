package service

import (
	"strings"
	"unicode"

	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/repository"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// FoodCacheService gerencia a lógica de cache de dados nutricionais de alimentos.
// Permite reutilizar informações já obtidas da OpenAI para economizar tokens.
type FoodCacheService struct {
	foodCacheRepo *repository.FoodCacheRepository
}

// NewFoodCacheService cria uma nova instância do serviço de cache de alimentos
func NewFoodCacheService(repo *repository.FoodCacheRepository) *FoodCacheService {
	return &FoodCacheService{
		foodCacheRepo: repo,
	}
}

// NormalizeName normaliza o nome do alimento: converte para minúsculas,
// remove espaços extras e remove acentos para busca consistente
func (s *FoodCacheService) NormalizeName(name string) string {
	// Remove espaços no início e fim
	name = strings.TrimSpace(name)

	// Converte para minúsculas
	name = strings.ToLower(name)

	// Remove acentos usando normalização Unicode NFD + remoção de combining marks
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, name)
	if err != nil {
		// Se falhar a remoção de acentos, retorna apenas em minúsculas
		return name
	}

	// Remove espaços duplicados
	fields := strings.Fields(result)
	return strings.Join(fields, " ")
}

// GetFromCache busca um alimento no cache pelo nome normalizado
func (s *FoodCacheService) GetFromCache(name string) (*model.FoodCache, error) {
	normalized := s.NormalizeName(name)
	return s.foodCacheRepo.FindByName(normalized)
}

// SaveToCache salva os dados nutricionais de um alimento no cache.
// Os valores são armazenados por unidade para facilitar recálculos futuros.
func (s *FoodCacheService) SaveToCache(food model.Food) error {
	normalized := s.NormalizeName(food.Name)

	// Calcula valores por unidade para armazenamento proporcional
	var perUnitProtein, perUnitCarbs, perUnitFat, perUnitCalories float64
	if food.Quantity > 0 {
		perUnitProtein = food.Protein / food.Quantity
		perUnitCarbs = food.Carbs / food.Quantity
		perUnitFat = food.Fat / food.Quantity
		perUnitCalories = food.Calories / food.Quantity
	} else {
		// Se a quantidade for zero ou negativa, usa os valores absolutos
		perUnitProtein = food.Protein
		perUnitCarbs = food.Carbs
		perUnitFat = food.Fat
		perUnitCalories = food.Calories
	}

	cache := &model.FoodCache{
		NameNormalized:  normalized,
		NameDisplay:     food.Name,
		Unit:            food.Unit,
		PerUnitProtein:  roundTo1Decimal(perUnitProtein),
		PerUnitCarbs:    roundTo1Decimal(perUnitCarbs),
		PerUnitFat:      roundTo1Decimal(perUnitFat),
		PerUnitCalories: roundTo1Decimal(perUnitCalories),
		Source:          "openai",
	}

	return s.foodCacheRepo.Save(cache)
}
