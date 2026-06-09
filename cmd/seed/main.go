package main

import (
	"context"
	"fmt"
	"os"

	postgresrepo "github.com/godlew/homecoin/internal/adapter/repository/postgres"
	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/godlew/homecoin/internal/infrastructure/auth"
	"github.com/godlew/homecoin/internal/infrastructure/config"
	"github.com/godlew/homecoin/internal/infrastructure/logger"
	"github.com/godlew/homecoin/internal/infrastructure/postgres"
	householduc "github.com/godlew/homecoin/internal/usecase/household"
)

const (
	seedPassword   = "password123"
	householdName  = "The Apartment"
	householdCurr  = "USD"
	ownerEmail     = "alice@homecoin.test"
)

type seedPerson struct {
	Email       string
	DisplayName string
}

var seedPeople = []seedPerson{
	{Email: ownerEmail, DisplayName: "Alice"},
	{Email: "bob@homecoin.test", DisplayName: "Bob"},
	{Email: "carol@homecoin.test", DisplayName: "Carol"},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	ctx := context.Background()

	if cfg.AutoMigrate {
		if err := postgres.RunMigrations(cfg.DatabaseURL, log); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			os.Exit(1)
		}
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	userRepo := postgresrepo.NewUserRepo(pool)
	householdRepo := postgresrepo.NewHouseholdRepo(pool)
	categoryRepo := postgresrepo.NewCategoryRepo(pool)

	createHH := householduc.NewCreateUseCase(householdRepo, categoryRepo)
	joinHH := householduc.NewJoinUseCase(householdRepo)

	if owner, err := userRepo.GetByEmail(ctx, ownerEmail); err == nil {
		printExisting(ctx, userRepo, householdRepo, owner)
		return
	} else if err != domainerrors.ErrNotFound {
		fmt.Fprintf(os.Stderr, "lookup owner: %v\n", err)
		os.Exit(1)
	}

	users := make([]*entity.User, len(seedPeople))
	for i, p := range seedPeople {
		u, err := ensureUser(ctx, userRepo, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create user %s: %v\n", p.Email, err)
			os.Exit(1)
		}
		users[i] = u
	}

	out, err := createHH.Execute(ctx, householduc.CreateInput{
		UserID:   users[0].ID,
		Name:     householdName,
		Currency: householdCurr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create household: %v\n", err)
		os.Exit(1)
	}

	for _, u := range users[1:] {
		if _, err := joinHH.Execute(ctx, householduc.JoinInput{
			UserID:     u.ID,
			InviteCode: out.InviteCode,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "join household (%s): %v\n", u.Email, err)
			os.Exit(1)
		}
	}

	fmt.Println("Seeded database successfully.")
	fmt.Println()
	fmt.Printf("Household: %s (%s)\n", householdName, out.HouseholdID)
	fmt.Printf("Invite code: %s\n", out.InviteCode)
	fmt.Println()
	fmt.Println("Members (password for all: " + seedPassword + "):")
	for i, p := range seedPeople {
		role := "member"
		if i == 0 {
			role = "owner"
		}
		fmt.Printf("  %s — %s (%s)\n", p.Email, p.DisplayName, role)
	}
	fmt.Println()
	fmt.Println("Log in at http://localhost:8081/login")
}

func ensureUser(ctx context.Context, users *postgresrepo.UserRepo, p seedPerson) (*entity.User, error) {
	existing, err := users.GetByEmail(ctx, p.Email)
	if err == nil {
		return existing, nil
	}
	if err != domainerrors.ErrNotFound {
		return nil, err
	}

	email, err := valueobject.NewEmail(p.Email)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(seedPassword)
	if err != nil {
		return nil, err
	}

	u := &entity.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  p.DisplayName,
	}
	if err := users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func printExisting(ctx context.Context, users *postgresrepo.UserRepo, households *postgresrepo.HouseholdRepo, owner *entity.User) {
	member, err := households.GetMemberByUserID(ctx, owner.ID)
	if err != nil {
		fmt.Println("Seed users exist but household not found — delete seed users or use a fresh database.")
		os.Exit(1)
	}

	hh, err := households.GetByID(ctx, member.HouseholdID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "household: %v\n", err)
		os.Exit(1)
	}

	members, err := households.GetMembers(ctx, member.HouseholdID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "members: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed data already present.")
	fmt.Println()
	fmt.Printf("Household: %s (%s)\n", hh.Name, hh.ID)
	if hh.InviteCode != nil {
		fmt.Printf("Invite code: %s\n", *hh.InviteCode)
	}
	fmt.Printf("Members: %d\n", len(members))
	fmt.Println()
	fmt.Println("Accounts (password for all: " + seedPassword + "):")
	for _, m := range members {
		u, err := users.GetByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		fmt.Printf("  %s — %s (%s)\n", u.Email, u.DisplayName, m.Role)
	}
}
