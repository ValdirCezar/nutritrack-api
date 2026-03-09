package repository

import (
	"context"
	"errors"
	"time"

	"github.com/valdircezar/nutritrack-api/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FoodCacheRepository gerencia o cache de dados nutricionais de alimentos no MongoDB.
// Evita chamadas repetidas à API da OpenAI para alimentos já processados.
type FoodCacheRepository struct {
	collection *mongo.Collection
}

// NewFoodCacheRepository cria uma nova instância do repositório de cache de alimentos
// e configura o índice único no campo name_normalized
func NewFoodCacheRepository(db *mongo.Database) *FoodCacheRepository {
	repo := &FoodCacheRepository{
		collection: db.Collection("food_cache"),
	}

	// Cria índice único no nome normalizado para buscas rápidas e sem duplicatas
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nameIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "name_normalized", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, _ = repo.collection.Indexes().CreateOne(ctx, nameIndex)

	return repo
}

// FindByName busca um alimento no cache pelo nome normalizado
func (r *FoodCacheRepository) FindByName(nameNormalized string) (*model.FoodCache, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cache model.FoodCache
	err := r.collection.FindOne(ctx, bson.M{"name_normalized": nameNormalized}).Decode(&cache)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &cache, nil
}

// Save salva ou atualiza um alimento no cache usando upsert pelo nome normalizado.
// Se o alimento já existir, atualiza os dados nutricionais.
func (r *FoodCacheRepository) Save(cache *model.FoodCache) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"name_normalized": cache.NameNormalized}
	update := bson.M{
		"$set": bson.M{
			"name_display":     cache.NameDisplay,
			"unit":             cache.Unit,
			"per_unit_protein":  cache.PerUnitProtein,
			"per_unit_carbs":    cache.PerUnitCarbs,
			"per_unit_fat":      cache.PerUnitFat,
			"per_unit_calories": cache.PerUnitCalories,
			"source":           cache.Source,
		},
		"$setOnInsert": bson.M{
			"name_normalized": cache.NameNormalized,
			"created_at":      time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
