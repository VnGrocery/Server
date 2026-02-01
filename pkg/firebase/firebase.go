package firebase

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"
	firebaseapp "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"vngrocery/pkg/config"
)

type App struct {
	AuthVerifier *firebaseauth.Client
	Firestore    *gofirestore.Client
}

func NewApp(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx := context.Background()

	options := []option.ClientOption{
		option.WithCredentialsFile(cfg.FirebaseCredentialsFile),
	}

	fbConfig := &firebaseapp.Config{}
	if cfg.FirebaseProjectID != "" {
		fbConfig.ProjectID = cfg.FirebaseProjectID
	}

	app, err := firebaseapp.NewApp(ctx, fbConfig, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase Auth client: %w", err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %w", err)
	}

	return &App{
		AuthVerifier: authClient,
		Firestore:    firestoreClient,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.Firestore == nil {
		return nil
	}

	return a.Firestore.Close()
}
