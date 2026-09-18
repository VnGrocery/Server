package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type SettingsRepository struct{ collection *mongo.Collection }

func NewSettingsRepository(db *mongo.Database) *SettingsRepository {
	return &SettingsRepository{collection: db.Collection(settingsCollection)}
}

// Get returns the runtime settings, or the defaults when nobody has saved any.
//
// A missing document is the normal state of a fresh install, not an error the
// caller has to handle - the rate limits have to resolve to a number either
// way.
func (r *SettingsRepository) Get(ctx context.Context) (domain.RuntimeSettings, error) {
	settings, err := getByID[domain.RuntimeSettings](ctx, r.collection, domain.RuntimeSettingsID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.DefaultRuntimeSettings(), nil
	}
	if err != nil {
		return domain.RuntimeSettings{}, err
	}
	return settings.Normalized(), nil
}

func (r *SettingsRepository) Save(ctx context.Context, settings domain.RuntimeSettings) error {
	return saveByID(ctx, r.collection, domain.RuntimeSettingsID, settings)
}
