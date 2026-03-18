package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrInvalidDefaultCurrency = errors.New("default_currency_code must be one of: EUR, USD, RUB")
var ErrInvalidNickname = errors.New("nickname must contain 3..32 chars: a-z, 0-9, _")
var ErrNicknameAlreadyExists = errors.New("nickname already exists")
var ErrInvalidName = errors.New("first_name and last_name are required")
var ErrCurrentPasswordInvalid = errors.New("current password is invalid")
var ErrWeakPassword = errors.New("new password must be at least 6 characters")

var nicknamePattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

type Service struct {
	q   *sqlc.Queries
	jwt *JWTManager
}

func NewService(q *sqlc.Queries, jwt *JWTManager) *Service {
	return &Service{q: q, jwt: jwt}
}

type LoginResult struct {
	Token string    `json:"token"`
	User  sqlc.User `json:"user"`
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	token, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, User: mapGetUserByEmailRow(user)}, nil
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName, nickname string) (LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	fullName := strings.TrimSpace(strings.TrimSpace(firstName + " " + lastName))
	if fullName == "" {
		fullName = firstName
	}
	normalizedNickname, err := normalizeNickname(nickname)
	if err != nil {
		return LoginResult{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return LoginResult{}, err
	}

	user, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(passwordHash),
		FullName:     fullName,
		FirstName:    firstName,
		LastName:     lastName,
		Nickname:     normalizedNickname,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "nickname") {
				return LoginResult{}, ErrNicknameAlreadyExists
			}
			return LoginResult{}, ErrEmailAlreadyExists
		}
		return LoginResult{}, err
	}

	token, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, User: mapCreateUserRow(user)}, nil
}

func (s *Service) Me(ctx context.Context, userID string) (sqlc.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return sqlc.User{}, err
	}
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return sqlc.User{}, err
	}
	return mapGetUserByIDRow(user), nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, firstName, lastName, nickname, defaultCurrencyCode string) (sqlc.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return sqlc.User{}, err
	}
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	fullName := strings.TrimSpace(strings.TrimSpace(firstName + " " + lastName))
	if fullName == "" {
		return sqlc.User{}, ErrInvalidName
	}
	normalizedNickname, err := normalizeNickname(nickname)
	if err != nil {
		return sqlc.User{}, err
	}
	normalizedCurrency, err := normalizeCurrencyCode(defaultCurrencyCode)
	if err != nil {
		return sqlc.User{}, err
	}

	user, err := s.q.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:                  id,
		FullName:            fullName,
		FirstName:           firstName,
		LastName:            lastName,
		Nickname:            normalizedNickname,
		DefaultCurrencyCode: normalizedCurrency,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "nickname") {
				return sqlc.User{}, ErrNicknameAlreadyExists
			}
		}
		return sqlc.User{}, err
	}
	return mapUpdateUserProfileRow(user), nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, defaultCurrencyCode string) (sqlc.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return sqlc.User{}, err
	}
	normalized, err := normalizeCurrencyCode(defaultCurrencyCode)
	if err != nil {
		return sqlc.User{}, err
	}
	user, err := s.q.UpdateUserDefaultCurrency(ctx, sqlc.UpdateUserDefaultCurrencyParams{
		ID:                  id,
		DefaultCurrencyCode: normalized,
	})
	if err != nil {
		return sqlc.User{}, err
	}
	return mapUpdateUserDefaultCurrencyRow(user), nil
}

func (s *Service) UpdateEmail(ctx context.Context, userID, currentPassword, newEmail string) (LoginResult, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return LoginResult{}, err
	}

	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return LoginResult{}, ErrCurrentPasswordInvalid
	}

	email := strings.TrimSpace(strings.ToLower(newEmail))
	updated, err := s.q.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{
		ID:    id,
		Email: email,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return LoginResult{}, ErrEmailAlreadyExists
		}
		return LoginResult{}, err
	}
	token, err := s.jwt.Generate(updated.ID, updated.Email)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, User: mapUpdateUserEmailRow(updated)}, nil
}

func (s *Service) UpdatePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(newPassword)) < 6 {
		return ErrWeakPassword
	}

	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return ErrCurrentPasswordInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.q.UpdateUserPasswordHash(ctx, sqlc.UpdateUserPasswordHashParams{
		ID:           id,
		PasswordHash: string(hash),
	})
}

func (s *Service) UpdateAvatarPath(ctx context.Context, userID string, avatarPath *string) (sqlc.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return sqlc.User{}, err
	}
	user, err := s.q.UpdateUserAvatarPath(ctx, sqlc.UpdateUserAvatarPathParams{
		ID:         id,
		AvatarPath: avatarPath,
	})
	if err != nil {
		return sqlc.User{}, err
	}
	return mapUpdateUserAvatarPathRow(user), nil
}

func mapGetUserByEmailRow(row sqlc.GetUserByEmailRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapCreateUserRow(row sqlc.CreateUserRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapGetUserByIDRow(row sqlc.GetUserByIDRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapUpdateUserProfileRow(row sqlc.UpdateUserProfileRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapUpdateUserDefaultCurrencyRow(row sqlc.UpdateUserDefaultCurrencyRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapUpdateUserEmailRow(row sqlc.UpdateUserEmailRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func mapUpdateUserAvatarPathRow(row sqlc.UpdateUserAvatarPathRow) sqlc.User {
	return sqlc.User{
		ID:                  row.ID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		FullName:            row.FullName,
		CreatedAt:           row.CreatedAt,
		DefaultCurrencyCode: row.DefaultCurrencyCode,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		Nickname:            row.Nickname,
		AvatarPath:          row.AvatarPath,
	}
}

func normalizeCurrencyCode(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "EUR", "USD", "RUB":
		return normalized, nil
	default:
		return "", ErrInvalidDefaultCurrency
	}
}

func normalizeNickname(value string) (string, error) {
	nickname := strings.ToLower(strings.TrimSpace(value))
	if !nicknamePattern.MatchString(nickname) {
		return "", ErrInvalidNickname
	}
	return nickname, nil
}
