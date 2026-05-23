package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	firestorerepo "vngrocery/internal/repository/firestore"
	mongorepo "vngrocery/internal/repository/mongo"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
	mongodbpkg "vngrocery/pkg/mongodb"
)

type userRepository interface {
	Save(ctx context.Context, user domain.User) error
	List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error)
}

type migrationStats struct {
	Total      int
	Changed    int
	Skipped    int
	Admins     int
	Users      int
	Legacy     map[string]int
	Unknown    map[string]int
	WouldWrite bool
}

func main() {
	apply := flag.Bool("apply", false, "apply updates; default is dry-run")
	mapUnknown := flag.Bool("map-unknown", false, "map empty or unknown roles to user when applying")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	var users userRepository
	var closeFn func() error

	if cfg.UseMongo() {
		app, err := mongodbpkg.NewApp(cfg)
		if err != nil {
			log.Fatalf("failed to initialize Mongo app: %v", err)
		}
		users = mongorepo.NewUserRepository(app.Database)
		closeFn = app.Close
	} else {
		app, err := firebasepkg.NewApp(cfg)
		if err != nil {
			log.Fatalf("failed to initialize Firebase app: %v", err)
		}
		users = firestorerepo.NewUserRepository(app.Firestore)
		closeFn = app.Close
	}
	defer func() {
		if closeFn != nil {
			if err := closeFn(); err != nil {
				log.Printf("failed to close app: %v", err)
			}
		}
	}()

	stats, err := migrateUserRoles(ctx, users, *apply, *mapUnknown)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	printStats(stats)
}

func migrateUserRoles(ctx context.Context, users userRepository, apply, mapUnknown bool) (migrationStats, error) {
	all, err := users.List(ctx, repository.UserListFilter{})
	if err != nil {
		return migrationStats{}, err
	}

	stats := migrationStats{
		Total:      len(all),
		Legacy:     map[string]int{},
		Unknown:    map[string]int{},
		WouldWrite: apply,
	}
	now := time.Now().UTC()
	for _, user := range all {
		role := strings.ToLower(strings.TrimSpace(user.Role))
		targetRole, shouldChange, known := normalizeLegacyRole(role, mapUnknown)
		switch targetRole {
		case domain.RoleAdmin:
			stats.Admins++
		case domain.RoleUser:
			stats.Users++
		}
		if role == "seller" || role == "buyer" {
			stats.Legacy[role]++
		}
		if !known {
			stats.Unknown[role]++
		}
		if !shouldChange {
			stats.Skipped++
			continue
		}
		stats.Changed++
		if !apply {
			continue
		}
		user.Role = targetRole
		user.Version++
		user.UpdatedAt = now
		if err := users.Save(ctx, user); err != nil {
			return stats, fmt.Errorf("save user %s: %w", user.UserID, err)
		}
	}
	return stats, nil
}

func normalizeLegacyRole(role string, mapUnknown bool) (target string, shouldChange bool, known bool) {
	switch role {
	case domain.RoleAdmin, domain.RoleUser:
		return role, false, true
	case "seller", "buyer":
		return domain.RoleUser, true, true
	case "":
		return domain.RoleUser, mapUnknown, false
	default:
		return domain.RoleUser, mapUnknown, false
	}
}

func printStats(stats migrationStats) {
	mode := "dry-run"
	if stats.WouldWrite {
		mode = "apply"
	}
	fmt.Printf("User role migration mode: %s\n", mode)
	fmt.Printf("Total users: %d\n", stats.Total)
	fmt.Printf("Changed: %d\n", stats.Changed)
	fmt.Printf("Skipped: %d\n", stats.Skipped)
	fmt.Printf("Target admins: %d\n", stats.Admins)
	fmt.Printf("Target users: %d\n", stats.Users)
	fmt.Printf("Legacy seller: %d\n", stats.Legacy["seller"])
	fmt.Printf("Legacy buyer: %d\n", stats.Legacy["buyer"])
	if len(stats.Unknown) > 0 {
		fmt.Println("Unknown roles:")
		for role, count := range stats.Unknown {
			fmt.Printf("- %q: %d\n", role, count)
		}
	}
}
