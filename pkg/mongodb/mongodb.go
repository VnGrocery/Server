package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"vngrocery/pkg/config"
)

type App struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewApp(cfg config.Config) (*App, error) {
	if !cfg.UseMongo() {
		return nil, fmt.Errorf("mongodb is disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect MongoDB: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &App{
		Client:   client,
		Database: client.Database(cfg.MongoDatabase),
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.Client == nil {
		return nil
	}
	return a.Client.Disconnect(context.Background())
}
