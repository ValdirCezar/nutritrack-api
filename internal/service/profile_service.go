package service

import (
	"errors"
	"math"

	"github.com/valdircezar/nutritrack-api/internal/model"
	"github.com/valdircezar/nutritrack-api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProfileService contém a lógica de negócio para gerenciamento de perfis nutricionais
type ProfileService struct {
	profileRepo *repository.ProfileRepository
}

// NewProfileService cria uma nova instância do serviço de perfis
func NewProfileService(repo *repository.ProfileRepository) *ProfileService {
	return &ProfileService{
		profileRepo: repo,
	}
}

// CreateOrUpdate valida os dados do perfil, calcula TMB, TDEE e macros,
// e salva ou atualiza o perfil do usuário no banco de dados
func (s *ProfileService) CreateOrUpdate(userID primitive.ObjectID, req model.ProfileRequest) (*model.Profile, error) {
	// Validação dos campos de entrada
	if err := validateProfileRequest(req); err != nil {
		return nil, err
	}

	// Calcula TMB usando fórmula de Mifflin-St Jeor
	tmb := calculateTMB(req.Weight, req.Height, req.Age, req.Sex)

	// Calcula TDEE = TMB × fator de atividade
	activityFactor, ok := model.ActivityFactors[req.ActivityLevel]
	if !ok {
		return nil, errors.New("nível de atividade inválido")
	}
	tdee := tmb * activityFactor

	// Ajusta calorias diárias: usa meta de peso personalizada se definida, senão porcentagem padrão
	var dailyCalories float64
	if req.TargetWeight > 0 && req.TargetWeeks > 0 {
		dailyCalories = adjustCaloriesForWeightGoal(tdee, req.Weight, req.TargetWeight, req.TargetWeeks)
	} else {
		dailyCalories = adjustCaloriesForGoal(tdee, req.Goal)
	}

	// Calcula macronutrientes com base no objetivo e peso corporal
	protein, fat := calculateMacrosByGoal(req.Weight, req.Goal)
	carbs := calculateCarbs(dailyCalories, protein, fat)

	// Arredonda todos os valores para 1 casa decimal
	profile := &model.Profile{
		UserID:        userID,
		Weight:        req.Weight,
		Height:        req.Height,
		Age:           req.Age,
		Sex:           req.Sex,
		ActivityLevel: req.ActivityLevel,
		Goal:          req.Goal,
		TargetWeight:  req.TargetWeight,
		TargetWeeks:   req.TargetWeeks,
		TMB:           roundTo1Decimal(tmb),
		TDEE:          roundTo1Decimal(tdee),
		DailyCalories: roundTo1Decimal(dailyCalories),
		DailyProtein:  roundTo1Decimal(protein),
		DailyCarbs:    roundTo1Decimal(carbs),
		DailyFat:      roundTo1Decimal(fat),
	}

	// Salva ou atualiza o perfil no banco de dados (upsert por user_id)
	if err := s.profileRepo.Upsert(profile); err != nil {
		return nil, err
	}

	// Recarrega o perfil completo do banco para retornar com todos os campos
	saved, err := s.profileRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// GetProfile busca o perfil de um usuário pelo seu ID
func (s *ProfileService) GetProfile(userID primitive.ObjectID) (*model.Profile, error) {
	return s.profileRepo.FindByUserID(userID)
}

// validateProfileRequest valida todos os campos do request de perfil
func validateProfileRequest(req model.ProfileRequest) error {
	// Peso: 30 a 300 kg
	if req.Weight < 30 || req.Weight > 300 {
		return errors.New("peso deve estar entre 30 e 300 kg")
	}

	// Altura: 100 a 250 cm
	if req.Height < 100 || req.Height > 250 {
		return errors.New("altura deve estar entre 100 e 250 cm")
	}

	// Idade: 10 a 120 anos
	if req.Age < 10 || req.Age > 120 {
		return errors.New("idade deve estar entre 10 e 120 anos")
	}

	// Sexo: deve ser "male" ou "female"
	if req.Sex != model.SexMale && req.Sex != model.SexFemale {
		return errors.New("sexo deve ser 'male' ou 'female'")
	}

	// Nível de atividade: deve ser um dos valores válidos
	if _, ok := model.ActivityFactors[req.ActivityLevel]; !ok {
		return errors.New("nível de atividade inválido. Valores aceitos: sedentary, light, moderate, very, extreme")
	}

	// Peso meta e prazo (opcionais): se um for informado, o outro também deve ser
	if req.TargetWeight > 0 || req.TargetWeeks > 0 {
		if req.TargetWeight <= 0 {
			return errors.New("peso meta é obrigatório quando o prazo é informado")
		}
		if req.TargetWeeks <= 0 {
			return errors.New("prazo em semanas é obrigatório quando o peso meta é informado")
		}
		if req.TargetWeight < 30 || req.TargetWeight > 300 {
			return errors.New("peso meta deve estar entre 30 e 300 kg")
		}
		if req.TargetWeeks < 1 || req.TargetWeeks > 104 {
			return errors.New("prazo deve estar entre 1 e 104 semanas (2 anos)")
		}
	}

	// Objetivo: deve ser um dos valores válidos
	validGoals := map[string]bool{
		model.GoalHypertrophy:   true,
		model.GoalWeightGain:    true,
		model.GoalWeightLoss:    true,
		model.GoalRecomposition: true,
		model.GoalMaintenance:   true,
	}
	if !validGoals[req.Goal] {
		return errors.New("objetivo inválido. Valores aceitos: hypertrophy, weight_gain, weight_loss, recomposition, maintenance")
	}

	return nil
}

// calculateTMB calcula a Taxa Metabólica Basal usando a fórmula de Mifflin-St Jeor
// Homem: (10 × peso) + (6.25 × altura) - (5 × idade) + 5
// Mulher: (10 × peso) + (6.25 × altura) - (5 × idade) - 161
func calculateTMB(weight, height float64, age int, sex string) float64 {
	base := (10 * weight) + (6.25 * height) - (5 * float64(age))
	if sex == model.SexMale {
		return base + 5
	}
	return base - 161
}

// adjustCaloriesForGoal aplica o multiplicador do objetivo sobre o TDEE
// hypertrophy: +12% (ponto médio de 10-15%)
// weight_gain: +20%
// weight_loss: -18% (ponto médio de 15-20% de déficit)
// recomposition: -5%
// maintenance: sem ajuste
func adjustCaloriesForGoal(tdee float64, goal string) float64 {
	switch goal {
	case model.GoalHypertrophy:
		return tdee * 1.12
	case model.GoalWeightGain:
		return tdee * 1.20
	case model.GoalWeightLoss:
		return tdee * 0.82
	case model.GoalRecomposition:
		return tdee * 0.95
	case model.GoalMaintenance:
		return tdee * 1.0
	default:
		return tdee
	}
}

// adjustCaloriesForWeightGoal calcula o ajuste calórico diário com base na meta de peso e prazo.
// Regra: 1 kg de peso corporal ≈ 7700 kcal
// Limites de segurança: máximo ±1000 kcal/dia, mínimo 1200 kcal/dia
func adjustCaloriesForWeightGoal(tdee, currentWeight, targetWeight float64, targetWeeks int) float64 {
	// Diferença de peso (positivo = ganho, negativo = perda)
	weightDiff := targetWeight - currentWeight

	// Total de calorias necessárias para alcançar a meta
	totalCalories := weightDiff * 7700

	// Ajuste diário = total ÷ (semanas × 7 dias)
	dailyAdjustment := totalCalories / float64(targetWeeks*7)

	// Limites de segurança: máximo ±1000 kcal/dia
	if dailyAdjustment > 1000 {
		dailyAdjustment = 1000
	}
	if dailyAdjustment < -1000 {
		dailyAdjustment = -1000
	}

	dailyCalories := tdee + dailyAdjustment

	// Piso de segurança: nunca abaixo de 1200 kcal/dia
	if dailyCalories < 1200 {
		dailyCalories = 1200
	}

	return dailyCalories
}

// calculateMacrosByGoal retorna a quantidade de proteína e gordura (em gramas)
// com base no peso corporal e no objetivo do usuário
func calculateMacrosByGoal(weight float64, goal string) (protein, fat float64) {
	switch goal {
	case model.GoalHypertrophy:
		protein = 2.0 * weight
		fat = 0.8 * weight
	case model.GoalWeightGain:
		protein = 1.8 * weight
		fat = 0.9 * weight
	case model.GoalWeightLoss:
		protein = 2.2 * weight
		fat = 0.7 * weight
	case model.GoalRecomposition:
		protein = 2.0 * weight
		fat = 0.8 * weight
	case model.GoalMaintenance:
		protein = 1.8 * weight
		fat = 0.8 * weight
	default:
		protein = 1.8 * weight
		fat = 0.8 * weight
	}
	return protein, fat
}

// calculateCarbs calcula os carboidratos restantes com base nas calorias totais
// Carbs = (DailyCalories - (proteína×4 + gordura×9)) / 4
func calculateCarbs(dailyCalories, protein, fat float64) float64 {
	caloriesFromProtein := protein * 4
	caloriesFromFat := fat * 9
	remainingCalories := dailyCalories - caloriesFromProtein - caloriesFromFat
	if remainingCalories < 0 {
		return 0
	}
	return remainingCalories / 4
}

// roundTo1Decimal arredonda um valor para 1 casa decimal
func roundTo1Decimal(value float64) float64 {
	return math.Round(value*10) / 10
}
