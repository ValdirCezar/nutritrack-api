package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Food representa um alimento individual dentro de uma refeição
type Food struct {
	Name     string  `bson:"name" json:"name"`
	Quantity float64 `bson:"quantity" json:"quantity"`
	Unit     string  `bson:"unit" json:"unit"`
	Protein  float64 `bson:"protein" json:"protein"`
	Carbs    float64 `bson:"carbs" json:"carbs"`
	Fat      float64 `bson:"fat" json:"fat"`
	Calories float64 `bson:"calories" json:"calories"`
}

// NutrientTotals totaliza macronutrientes
type NutrientTotals struct {
	Protein  float64 `bson:"protein" json:"protein"`
	Carbs    float64 `bson:"carbs" json:"carbs"`
	Fat      float64 `bson:"fat" json:"fat"`
	Calories float64 `bson:"calories" json:"calories"`
}

// Meal representa uma refeição registrada pelo usuário
type Meal struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Date        string             `bson:"date" json:"date"` // formato YYYY-MM-DD
	Description string             `bson:"description" json:"description"`
	Foods       []Food             `bson:"foods" json:"foods"`
	Totals      NutrientTotals     `bson:"totals" json:"totals"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// MealRequest dados enviados pelo frontend para registrar uma refeição
type MealRequest struct {
	Description string `json:"description"`
	Date        string `json:"date,omitempty"`
}

// DashboardResponse resposta consolidada do dashboard diário
type DashboardResponse struct {
	Date       string         `json:"date"`
	Goals      NutrientTotals `json:"goals"`
	Consumed   NutrientTotals `json:"consumed"`
	Remaining  NutrientTotals `json:"remaining"`
	Percentage NutrientTotals `json:"percentage"`
	Meals      []Meal         `json:"meals"`
}
