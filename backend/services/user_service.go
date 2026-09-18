package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"livepoll/config"
	"livepoll/models"
	"livepoll/repositories"
	"livepoll/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository, cfg *config.Config) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Signup(ctx context.Context, name, email, password string) (*models.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if name == "" || len(name) < 2 {
		return nil, errors.New("name is required")
	}
	if !isValidEmail(email) {
		return nil, errors.New("invalid email")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}
	if exists, err := s.repo.CheckExistsByEmail(ctx, email); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.New("email already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, email, password string, cfg *config.Config) (*models.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	token, err := utils.GenerateJWT(cfg.JWTSecret, user.ID.Hex(), user.Email, cfg.JWTExpiration)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func isValidEmail(email string) bool {
	pattern := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	return pattern.MatchString(email)
}

func MustObjectID(value string) primitive.ObjectID {
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		panic(err)
	}
	return id
}
