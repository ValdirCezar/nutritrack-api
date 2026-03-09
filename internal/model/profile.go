package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Constantes para níveis de atividade
const (
	ActivitySedentary  = "sedentary"
	ActivityLight      = "light"
	ActivityModerate   = "moderate"
	ActivityVery       = "very"
	ActivityExtreme    = "extreme"
)

// Constantes para objetivos físicos
const (
	GoalHypertrophy   = "hypertrophy"
	GoalWeightGain    = "weight_gain"
	GoalWeightLoss    = "weight_loss"
	GoalRecomposition = "recomposition"
	GoalMaintenance   = "maintenance"
)

// Constantes para sexo
const (
	SexMale   = "male"
	SexFemale = "female"
)

// ActivityFactors mapeia nível de atividade para multiplicador
var ActivityFactors = map[string]float64{
	ActivitySedentary: 1.2,
	ActivityLight:     1.375,
	ActivityModerate:  1.55,
	ActivityVery:      1.725,
	ActivityExtreme:   1.9,
}

// Profile armazena o perfil físico e metas calculadas do usuário
type Profile struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	Weight        float64            `bson:"weight" json:"weight"`               // kg atual
	Height        float64            `bson:"height" json:"height"`               // cm
	Age           int                `bson:"age" json:"age"`
	Sex           string             `bson:"sex" json:"sex"`                     // male, female
	ActivityLevel string             `bson:"activity_level" json:"activity_level"` // sedentary, light, moderate, very, extreme
	Goal          string             `bson:"goal" json:"goal"`                   // hypertrophy, weight_gain, weight_loss, recomposition, maintenance
	TargetWeight  float64            `bson:"target_weight" json:"target_weight"` // kg desejado (0 = usar porcentagem padrão)
	TargetWeeks   int                `bson:"target_weeks" json:"target_weeks"`   // prazo em semanas (0 = usar porcentagem padrão)
	TMB           float64            `bson:"tmb" json:"tmb"`
	TDEE          float64            `bson:"tdee" json:"tdee"`
	DailyCalories float64            `bson:"daily_calories" json:"daily_calories"`
	DailyProtein  float64            `bson:"daily_protein" json:"daily_protein"`
	DailyCarbs    float64            `bson:"daily_carbs" json:"daily_carbs"`
	DailyFat      float64            `bson:"daily_fat" json:"daily_fat"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// ProfileRequest dados enviados pelo frontend no onboarding
type ProfileRequest struct {
	Weight        float64 `json:"weight"`
	Height        float64 `json:"height"`
	Age           int     `json:"age"`
	Sex           string  `json:"sex"`
	ActivityLevel string  `json:"activity_level"`
	Goal          string  `json:"goal"`
	TargetWeight  float64 `json:"target_weight"`  // opcional: peso desejado em kg
	TargetWeeks   int     `json:"target_weeks"`   // opcional: prazo em semanas
}
