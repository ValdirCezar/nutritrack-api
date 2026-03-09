package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FoodCache armazena dados nutricionais de alimentos já processados pela OpenAI
// para evitar chamadas repetidas e reduzir custo de tokens
type FoodCache struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NameNormalized  string             `bson:"name_normalized" json:"name_normalized"` // nome em lowercase sem acentos
	NameDisplay     string             `bson:"name_display" json:"name_display"`
	Unit            string             `bson:"unit" json:"unit"`
	PerUnitProtein  float64            `bson:"per_unit_protein" json:"per_unit_protein"`
	PerUnitCarbs    float64            `bson:"per_unit_carbs" json:"per_unit_carbs"`
	PerUnitFat      float64            `bson:"per_unit_fat" json:"per_unit_fat"`
	PerUnitCalories float64            `bson:"per_unit_calories" json:"per_unit_calories"`
	Source          string             `bson:"source" json:"source"` // "openai"
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}

// OpenAIFoodResponse formato de retorno da OpenAI para um alimento
type OpenAIFoodResponse struct {
	Foods  []Food         `json:"foods"`
	Totals NutrientTotals `json:"totals"`
}
