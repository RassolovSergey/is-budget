package projects

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

type ProjectListItem struct {
	sqlc.Project
	CurrentUserRole string `json:"current_user_role"`
}

type CategoryFolderSummary struct {
	Category                 string `json:"category"`
	ActiveProjectsCount      int    `json:"active_projects_count"`
	CompletedProjectsCount   int    `json:"completed_projects_count"`
	HistoricalCampaignsCount int    `json:"historical_campaigns_count"`
}

type historicalCampaignCategoryCount struct {
	Category string
	Count    int
}

type ProjectInviteView struct {
	ID                  uuid.UUID  `json:"id"`
	ProjectID           uuid.UUID  `json:"project_id"`
	Token               string     `json:"token"`
	InvitedRole         string     `json:"invited_role"`
	InvitedBy           uuid.UUID  `json:"invited_by"`
	InvitedUserID       *uuid.UUID `json:"invited_user_id"`
	InvitedUserEmail    *string    `json:"invited_user_email"`
	InvitedUserFullName *string    `json:"invited_user_full_name"`
	InvitedUserNickname *string    `json:"invited_user_nickname"`
	InvitedByFullName   *string    `json:"invited_by_full_name"`
	AcceptedBy          *uuid.UUID `json:"accepted_by"`
	AcceptedAt          *time.Time `json:"accepted_at"`
	DeclinedBy          *uuid.UUID `json:"declined_by"`
	DeclinedAt          *time.Time `json:"declined_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	CreatedAt           time.Time  `json:"created_at"`
	Status              string     `json:"status"`
}

type IncomingProjectInvite struct {
	ID                  uuid.UUID `json:"id"`
	Token               string    `json:"token"`
	ProjectID           uuid.UUID `json:"project_id"`
	ProjectName         string    `json:"project_name"`
	ProjectDescription  string    `json:"project_description"`
	ProjectStatus       string    `json:"project_status"`
	ProjectCurrencyCode string    `json:"project_currency_code"`
	InvitedRole         string    `json:"invited_role"`
	InvitedBy           uuid.UUID `json:"invited_by"`
	InvitedByFullName   *string   `json:"invited_by_full_name"`
	ExpiresAt           time.Time `json:"expires_at"`
	CreatedAt           time.Time `json:"created_at"`
}

const (
	ProjectStatusPreparation = "preparation"
	ProjectStatusActive      = "active"
	ProjectStatusCompleted   = "completed"
)

var (
	ErrLastPM                               = errors.New("cannot remove last PM from project")
	ErrPMCannotLeaveProject                 = errors.New("project administrator cannot leave project; delete project instead")
	ErrProjectAdministratorFixed            = errors.New("project administrator is fixed to the project creator")
	ErrInvalidProjectRole                   = errors.New("invalid role")
	ErrInvalidProjectDuration               = errors.New("duration_days must be between 3 and 30")
	ErrInvalidProjectStatus                 = errors.New("invalid project status")
	ErrInvalidProjectCurrency               = errors.New("currency_code must be one of: EUR, USD, RUB")
	ErrInvalidSecondaryProjectCurrency      = errors.New("secondary_currency_code must be one of: EUR, USD, RUB")
	ErrSecondaryCurrencyMustDiffer          = errors.New("secondary_currency_code must be different from currency_code")
	ErrInvalidProjectCategory               = errors.New("category must be between 2 and 80 characters")
	ErrProjectDurationEditOnlyInPreparation = errors.New("duration_days can be changed only while project is in preparation status")
	ErrProjectActivationOnlyInPreparation   = errors.New("project can be activated only while status is preparation")
	ErrInvalidInviteTTL                     = errors.New("expires_days must be between 1 and 30")
	ErrInviteNotFound                       = errors.New("invite not found")
	ErrInviteExpired                        = errors.New("invite expired")
	ErrInviteAlreadyAccepted                = errors.New("invite already accepted")
	ErrInviteAlreadyDeclined                = errors.New("invite already declined")
	ErrInviteNotForUser                     = errors.New("invite is not addressed to this user")
	ErrInviteAlreadyPending                 = errors.New("invite already sent and pending confirmation")
	ErrInviteLinksDisabled                  = errors.New("invite links are disabled; specify a concrete recipient")
	ErrMemberAlreadyExists                  = errors.New("member already exists")
	ErrInviteCannotBeRevoked                = errors.New("only pending invite can be revoked")
)

func NewService(q *sqlc.Queries, pool *pgxpool.Pool) *Service {
	return &Service{q: q, pool: pool}
}

func (s *Service) ListProjects(ctx context.Context, userID uuid.UUID) ([]ProjectListItem, error) {
	if err := s.q.CompleteProjectsForUserIfDue(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListProjectsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ProjectListItem{
			Project:         mapListProjectRow(row),
			CurrentUserRole: string(row.CurrentUserRole),
		})
	}
	return items, nil
}

func (s *Service) ListCategoryFolders(ctx context.Context, userID uuid.UUID) ([]CategoryFolderSummary, error) {
	if err := s.q.CompleteProjectsForUserIfDue(ctx, userID); err != nil {
		return nil, err
	}

	projects, err := s.q.ListProjectsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	historicalCounts, err := s.listHistoricalCampaignCountsByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}

	return buildCategoryFolderSummaries(projects, historicalCounts), nil
}

func (s *Service) listHistoricalCampaignCountsByOwner(ctx context.Context, userID uuid.UUID) ([]historicalCampaignCategoryCount, error) {
	rows, err := s.pool.Query(ctx, `
SELECT COALESCE(NULLIF(TRIM(category), ''), 'Общее') AS category,
       COUNT(*)::integer AS historical_campaigns_count
FROM historical_campaigns
WHERE owner_user_id = $1
GROUP BY 1
ORDER BY 1
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]historicalCampaignCategoryCount, 0)
	for rows.Next() {
		var item historicalCampaignCategoryCount
		if err := rows.Scan(&item.Category, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func buildCategoryFolderSummaries(projects []sqlc.ListProjectsForUserRow, historicalCounts []historicalCampaignCategoryCount) []CategoryFolderSummary {
	byCategory := make(map[string]*CategoryFolderSummary)

	ensure := func(category string) *CategoryFolderSummary {
		normalized := canonicalProjectCategory(category)
		item, ok := byCategory[normalized]
		if !ok {
			item = &CategoryFolderSummary{Category: normalized}
			byCategory[normalized] = item
		}
		return item
	}

	for _, project := range projects {
		item := ensure(project.Category)
		if project.Status == ProjectStatusCompleted {
			item.CompletedProjectsCount++
			continue
		}
		item.ActiveProjectsCount++
	}

	for _, count := range historicalCounts {
		item := ensure(count.Category)
		item.HistoricalCampaignsCount += count.Count
	}

	items := make([]CategoryFolderSummary, 0, len(byCategory))
	for _, item := range byCategory {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Category) < strings.ToLower(items[j].Category)
	})
	return items
}

func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, name, description string, durationDays int32, currencyCode, secondaryCurrencyCode, category string) (sqlc.Project, error) {
	if !isValidDurationDays(durationDays) {
		return sqlc.Project{}, ErrInvalidProjectDuration
	}
	normalizedCurrency, err := normalizeCurrencyCode(currencyCode)
	if err != nil {
		return sqlc.Project{}, err
	}
	normalizedSecondaryCurrency, err := normalizeSecondaryCurrencyCode(normalizedCurrency, secondaryCurrencyCode)
	if err != nil {
		return sqlc.Project{}, err
	}
	normalizedCategory, err := normalizeProjectCategory(category)
	if err != nil {
		return sqlc.Project{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return sqlc.Project{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	project, err := qtx.CreateProject(ctx, sqlc.CreateProjectParams{
		Name:                  strings.TrimSpace(name),
		Description:           strings.TrimSpace(description),
		DurationDays:          durationDays,
		CurrencyCode:          normalizedCurrency,
		SecondaryCurrencyCode: normalizedSecondaryCurrency,
		Category:              normalizedCategory,
		CreatedBy:             userID,
	})
	if err != nil {
		return sqlc.Project{}, err
	}

	if _, err := qtx.AddProjectMember(ctx, sqlc.AddProjectMemberParams{
		ProjectID: project.ID,
		UserID:    userID,
		Role:      "PM",
	}); err != nil {
		return sqlc.Project{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Project{}, err
	}
	return mapCreateProjectRow(project), nil
}

func (s *Service) GetProject(ctx context.Context, projectID uuid.UUID) (sqlc.Project, error) {
	if err := s.q.CompleteProjectIfDue(ctx, projectID); err != nil {
		return sqlc.Project{}, err
	}
	row, err := s.q.GetProjectByID(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, err
	}
	return mapGetProjectByIDRow(row), nil
}

func (s *Service) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	member, err := s.q.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
	if err != nil {
		return "", err
	}
	return string(member.Role), nil
}

func (s *Service) UpdateProject(ctx context.Context, projectID uuid.UUID, name, description string, durationDays int32, currencyCode, secondaryCurrencyCode, category string) (sqlc.Project, error) {
	if !isValidDurationDays(durationDays) {
		return sqlc.Project{}, ErrInvalidProjectDuration
	}
	normalizedCurrency, err := normalizeCurrencyCode(currencyCode)
	if err != nil {
		return sqlc.Project{}, err
	}
	normalizedSecondaryCurrency, err := normalizeSecondaryCurrencyCode(normalizedCurrency, secondaryCurrencyCode)
	if err != nil {
		return sqlc.Project{}, err
	}
	normalizedCategory, err := normalizeProjectCategory(category)
	if err != nil {
		return sqlc.Project{}, err
	}

	if err := s.q.CompleteProjectIfDue(ctx, projectID); err != nil {
		return sqlc.Project{}, err
	}

	currentProject, err := s.q.GetProjectByID(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, err
	}
	currentStatus, err := normalizeProjectStatus(currentProject.Status)
	if err != nil {
		return sqlc.Project{}, err
	}

	if currentStatus != ProjectStatusPreparation && durationDays != currentProject.DurationDays {
		return sqlc.Project{}, ErrProjectDurationEditOnlyInPreparation
	}

	row, err := s.q.UpdateProject(ctx, sqlc.UpdateProjectParams{
		ID:                    projectID,
		Name:                  strings.TrimSpace(name),
		Description:           strings.TrimSpace(description),
		DurationDays:          durationDays,
		CurrencyCode:          normalizedCurrency,
		SecondaryCurrencyCode: normalizedSecondaryCurrency,
		Category:              normalizedCategory,
	})
	if err != nil {
		return sqlc.Project{}, err
	}
	return mapUpdateProjectRow(row), nil
}

func (s *Service) ActivateProject(ctx context.Context, projectID uuid.UUID) (sqlc.Project, error) {
	if err := s.q.CompleteProjectIfDue(ctx, projectID); err != nil {
		return sqlc.Project{}, err
	}

	currentProject, err := s.q.GetProjectByID(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, err
	}

	currentStatus, err := normalizeProjectStatus(currentProject.Status)
	if err != nil {
		return sqlc.Project{}, err
	}
	if currentStatus != ProjectStatusPreparation {
		return sqlc.Project{}, ErrProjectActivationOnlyInPreparation
	}

	row, err := s.q.ActivateProject(ctx, projectID)
	if err != nil {
		return sqlc.Project{}, err
	}
	return mapActivateProjectRow(row), nil
}

func (s *Service) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	return s.q.DeleteProject(ctx, projectID)
}

func (s *Service) LeaveProject(ctx context.Context, projectID, userID uuid.UUID) error {
	memberList, err := s.q.ListProjectMembers(ctx, projectID)
	if err != nil {
		return err
	}
	var self *sqlc.ProjectMemberWithUser
	for i := range memberList {
		if memberList[i].UserID == userID {
			self = &memberList[i]
			break
		}
	}
	if self == nil {
		return pgx.ErrNoRows
	}
	if self.Role == "PM" {
		return ErrPMCannotLeaveProject
	}
	return s.q.DeleteProjectMemberByUser(ctx, sqlc.DeleteProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    userID,
	})
}

func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID) ([]sqlc.ProjectMemberWithUser, error) {
	return s.q.ListProjectMembers(ctx, projectID)
}

func (s *Service) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) (sqlc.ProjectMember, error) {
	role = strings.ToUpper(strings.TrimSpace(role))
	if err := sqlc.ValidateRole(role); err != nil {
		return sqlc.ProjectMember{}, ErrInvalidProjectRole
	}
	if role == "PM" {
		return sqlc.ProjectMember{}, ErrProjectAdministratorFixed
	}

	if _, err := s.q.GetUserByID(ctx, userID); err != nil {
		return sqlc.ProjectMember{}, err
	}

	return s.q.AddProjectMember(ctx, sqlc.AddProjectMemberParams{
		ProjectID: projectID,
		UserID:    userID,
		Role:      sqlc.ProjectRole(role),
	})
}

func (s *Service) InviteMember(ctx context.Context, projectID, invitedBy, invitedUserID uuid.UUID, role string, expiresDays int32) (sqlc.ProjectInvite, error) {
	role = strings.ToUpper(strings.TrimSpace(role))
	if err := sqlc.ValidateRole(role); err != nil {
		return sqlc.ProjectInvite{}, ErrInvalidProjectRole
	}
	if role == "PM" {
		return sqlc.ProjectInvite{}, ErrProjectAdministratorFixed
	}

	if _, err := s.q.GetUserByID(ctx, invitedUserID); err != nil {
		return sqlc.ProjectInvite{}, err
	}
	if _, err := s.q.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: projectID,
		UserID:    invitedUserID,
	}); err == nil {
		return sqlc.ProjectInvite{}, ErrMemberAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.ProjectInvite{}, err
	}

	if _, err := s.q.GetPendingProjectInviteForUserInProject(ctx, sqlc.GetPendingProjectInviteForUserInProjectParams{
		ProjectID:     projectID,
		InvitedUserID: &invitedUserID,
	}); err == nil {
		return sqlc.ProjectInvite{}, ErrInviteAlreadyPending
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.ProjectInvite{}, err
	}

	return s.CreateInvite(ctx, projectID, invitedBy, role, expiresDays, &invitedUserID)
}

func (s *Service) ResolveUserIDByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	user, err := s.q.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

func (s *Service) ResolveUserIDByNickname(ctx context.Context, nickname string) (uuid.UUID, error) {
	user, err := s.q.GetUserByNickname(ctx, strings.ToLower(strings.TrimSpace(nickname)))
	if err != nil {
		return uuid.Nil, err
	}
	return user.ID, nil
}

func (s *Service) CreateInvite(ctx context.Context, projectID, invitedBy uuid.UUID, role string, expiresDays int32, invitedUserID *uuid.UUID) (sqlc.ProjectInvite, error) {
	if invitedUserID == nil {
		return sqlc.ProjectInvite{}, ErrInviteLinksDisabled
	}

	role = strings.ToUpper(strings.TrimSpace(role))
	if err := sqlc.ValidateRole(role); err != nil {
		return sqlc.ProjectInvite{}, ErrInvalidProjectRole
	}
	if role == "PM" {
		return sqlc.ProjectInvite{}, ErrProjectAdministratorFixed
	}

	if expiresDays == 0 {
		expiresDays = 7
	}
	if expiresDays < 1 || expiresDays > 30 {
		return sqlc.ProjectInvite{}, ErrInvalidInviteTTL
	}

	token := uuid.NewString()
	expiresAt := time.Now().UTC().AddDate(0, 0, int(expiresDays))
	row, err := s.q.CreateProjectInvite(ctx, sqlc.CreateProjectInviteParams{
		ProjectID:     projectID,
		Token:         token,
		InvitedRole:   sqlc.ProjectRole(role),
		InvitedBy:     invitedBy,
		InvitedUserID: invitedUserID,
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		return sqlc.ProjectInvite{}, err
	}
	return mapCreateProjectInviteRow(row), nil
}

func (s *Service) ListInvites(ctx context.Context, projectID uuid.UUID) ([]ProjectInviteView, error) {
	rows, err := s.q.ListProjectInvitesWithUsersByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectInviteView, 0, len(rows))
	for _, row := range rows {
		if row.InvitedUserID == nil {
			continue
		}
		items = append(items, mapProjectInviteViewRow(row))
	}
	return items, nil
}

func (s *Service) ListIncomingInvites(ctx context.Context, userID uuid.UUID) ([]IncomingProjectInvite, error) {
	rows, err := s.q.ListIncomingPendingProjectInvitesForUser(ctx, &userID)
	if err != nil {
		return nil, err
	}
	items := make([]IncomingProjectInvite, 0, len(rows))
	for _, row := range rows {
		items = append(items, IncomingProjectInvite{
			ID:                  row.ID,
			Token:               row.Token,
			ProjectID:           row.ProjectID,
			ProjectName:         row.ProjectName,
			ProjectDescription:  row.ProjectDescription,
			ProjectStatus:       row.ProjectStatus,
			ProjectCurrencyCode: row.ProjectCurrencyCode,
			InvitedRole:         string(row.InvitedRole),
			InvitedBy:           row.InvitedBy,
			InvitedByFullName:   row.InvitedByFullName,
			ExpiresAt:           row.ExpiresAt,
			CreatedAt:           row.CreatedAt,
		})
	}
	return items, nil
}

type AcceptInviteResult struct {
	ProjectID uuid.UUID `json:"project_id"`
	Role      string    `json:"role"`
}

func (s *Service) AcceptInvite(ctx context.Context, token string, userID uuid.UUID) (AcceptInviteResult, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AcceptInviteResult{}, ErrInviteNotFound
	}
	invite, err := s.q.GetProjectInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AcceptInviteResult{}, ErrInviteNotFound
		}
		return AcceptInviteResult{}, err
	}
	if invite.AcceptedAt != nil {
		return AcceptInviteResult{}, ErrInviteAlreadyAccepted
	}
	if invite.DeclinedAt != nil {
		return AcceptInviteResult{}, ErrInviteAlreadyDeclined
	}
	if invite.InvitedUserID == nil {
		return AcceptInviteResult{}, ErrInviteLinksDisabled
	}
	if invite.InvitedUserID != nil && *invite.InvitedUserID != userID {
		return AcceptInviteResult{}, ErrInviteNotForUser
	}
	if invite.ExpiresAt.Before(time.Now().UTC()) {
		return AcceptInviteResult{}, ErrInviteExpired
	}
	if invite.InvitedRole == "PM" {
		return AcceptInviteResult{}, ErrProjectAdministratorFixed
	}

	_, err = s.q.GetProjectMemberByUser(ctx, sqlc.GetProjectMemberByUserParams{
		ProjectID: invite.ProjectID,
		UserID:    userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, addErr := s.q.AddProjectMember(ctx, sqlc.AddProjectMemberParams{
				ProjectID: invite.ProjectID,
				UserID:    userID,
				Role:      invite.InvitedRole,
			}); addErr != nil {
				return AcceptInviteResult{}, addErr
			}
		} else {
			return AcceptInviteResult{}, err
		}
	}

	if _, err := s.q.MarkProjectInviteAccepted(ctx, sqlc.MarkProjectInviteAcceptedParams{
		ID:         invite.ID,
		AcceptedBy: &userID,
		AcceptedAt: ptrTime(time.Now().UTC()),
	}); err != nil {
		return AcceptInviteResult{}, err
	}

	return AcceptInviteResult{
		ProjectID: invite.ProjectID,
		Role:      string(invite.InvitedRole),
	}, nil
}

func (s *Service) DeclineInvite(ctx context.Context, token string, userID uuid.UUID) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInviteNotFound
	}

	invite, err := s.q.GetProjectInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteNotFound
		}
		return err
	}
	if invite.AcceptedAt != nil {
		return ErrInviteAlreadyAccepted
	}
	if invite.DeclinedAt != nil {
		return ErrInviteAlreadyDeclined
	}
	if invite.InvitedUserID == nil {
		return ErrInviteLinksDisabled
	}
	if invite.InvitedUserID != nil && *invite.InvitedUserID != userID {
		return ErrInviteNotForUser
	}
	if invite.ExpiresAt.Before(time.Now().UTC()) {
		return ErrInviteExpired
	}

	_, err = s.q.MarkProjectInviteDeclined(ctx, sqlc.MarkProjectInviteDeclinedParams{
		ID:         invite.ID,
		DeclinedBy: &userID,
		DeclinedAt: ptrTime(time.Now().UTC()),
	})
	return err
}

func (s *Service) RevokeInvite(ctx context.Context, projectID, inviteID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM project_invites
		WHERE id = $1
		  AND project_id = $2
		  AND accepted_at IS NULL
		  AND declined_at IS NULL
	`, inviteID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteCannotBeRevoked
	}
	return nil
}

func (s *Service) DeleteMember(ctx context.Context, projectID, memberID uuid.UUID) error {
	project, err := s.q.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}

	memberList, err := s.q.ListProjectMembers(ctx, projectID)
	if err != nil {
		return err
	}
	for _, m := range memberList {
		if m.ID != memberID {
			continue
		}
		if m.Role == "PM" || m.UserID == project.CreatedBy {
			return ErrProjectAdministratorFixed
		}
		break
	}
	return s.q.DeleteProjectMember(ctx, sqlc.DeleteProjectMemberParams{ProjectID: projectID, ID: memberID})
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func mapProjectBase(
	id uuid.UUID,
	name, description string,
	durationDays int32,
	status string,
	activatedAt *time.Time,
	currencyCode string,
	secondaryCurrencyCode *string,
	category string,
	createdBy uuid.UUID,
	createdAt, updatedAt time.Time,
) sqlc.Project {
	return sqlc.Project{
		ID:                    id,
		Name:                  name,
		Description:           description,
		CreatedBy:             createdBy,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		DurationDays:          durationDays,
		Status:                status,
		ActivatedAt:           activatedAt,
		CurrencyCode:          currencyCode,
		Category:              category,
		SecondaryCurrencyCode: secondaryCurrencyCode,
	}
}

func mapListProjectRow(row sqlc.ListProjectsForUserRow) sqlc.Project {
	return mapProjectBase(
		row.ID,
		row.Name,
		row.Description,
		row.DurationDays,
		row.Status,
		row.ActivatedAt,
		row.CurrencyCode,
		row.SecondaryCurrencyCode,
		row.Category,
		row.CreatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapCreateProjectRow(row sqlc.CreateProjectRow) sqlc.Project {
	return mapProjectBase(
		row.ID,
		row.Name,
		row.Description,
		row.DurationDays,
		row.Status,
		row.ActivatedAt,
		row.CurrencyCode,
		row.SecondaryCurrencyCode,
		row.Category,
		row.CreatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapGetProjectByIDRow(row sqlc.GetProjectByIDRow) sqlc.Project {
	return mapProjectBase(
		row.ID,
		row.Name,
		row.Description,
		row.DurationDays,
		row.Status,
		row.ActivatedAt,
		row.CurrencyCode,
		row.SecondaryCurrencyCode,
		row.Category,
		row.CreatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapUpdateProjectRow(row sqlc.UpdateProjectRow) sqlc.Project {
	return mapProjectBase(
		row.ID,
		row.Name,
		row.Description,
		row.DurationDays,
		row.Status,
		row.ActivatedAt,
		row.CurrencyCode,
		row.SecondaryCurrencyCode,
		row.Category,
		row.CreatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapActivateProjectRow(row sqlc.ActivateProjectRow) sqlc.Project {
	return mapProjectBase(
		row.ID,
		row.Name,
		row.Description,
		row.DurationDays,
		row.Status,
		row.ActivatedAt,
		row.CurrencyCode,
		row.SecondaryCurrencyCode,
		row.Category,
		row.CreatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapCreateProjectInviteRow(row sqlc.CreateProjectInviteRow) sqlc.ProjectInvite {
	return sqlc.ProjectInvite{
		ID:            row.ID,
		ProjectID:     row.ProjectID,
		Token:         row.Token,
		InvitedRole:   row.InvitedRole,
		InvitedBy:     row.InvitedBy,
		AcceptedBy:    row.AcceptedBy,
		AcceptedAt:    row.AcceptedAt,
		ExpiresAt:     row.ExpiresAt,
		CreatedAt:     row.CreatedAt,
		InvitedUserID: row.InvitedUserID,
		DeclinedBy:    row.DeclinedBy,
		DeclinedAt:    row.DeclinedAt,
	}
}

func mapProjectInviteViewRow(row sqlc.ListProjectInvitesWithUsersByProjectRow) ProjectInviteView {
	status := "pending"
	if row.AcceptedAt != nil {
		status = "accepted"
	} else if row.DeclinedAt != nil {
		status = "declined"
	} else if row.ExpiresAt.Before(time.Now().UTC()) {
		status = "expired"
	}

	return ProjectInviteView{
		ID:                  row.ID,
		ProjectID:           row.ProjectID,
		Token:               row.Token,
		InvitedRole:         string(row.InvitedRole),
		InvitedBy:           row.InvitedBy,
		InvitedUserID:       row.InvitedUserID,
		InvitedUserEmail:    row.InvitedUserEmail,
		InvitedUserFullName: row.InvitedUserFullName,
		InvitedUserNickname: row.InvitedUserNickname,
		InvitedByFullName:   row.InvitedByFullName,
		AcceptedBy:          row.AcceptedBy,
		AcceptedAt:          row.AcceptedAt,
		DeclinedBy:          row.DeclinedBy,
		DeclinedAt:          row.DeclinedAt,
		ExpiresAt:           row.ExpiresAt,
		CreatedAt:           row.CreatedAt,
		Status:              status,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isValidDurationDays(durationDays int32) bool {
	return durationDays >= 3 && durationDays <= 30
}

func normalizeProjectStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case ProjectStatusPreparation, ProjectStatusActive, ProjectStatusCompleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidProjectStatus, status)
	}
}

func normalizeCurrencyCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		return "EUR", nil
	}
	switch normalized {
	case "EUR", "USD", "RUB":
		return normalized, nil
	default:
		return "", ErrInvalidProjectCurrency
	}
}

func normalizeSecondaryCurrencyCode(primary, secondary string) (*string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secondary))
	if normalized == "" {
		return nil, nil
	}
	switch normalized {
	case "EUR", "USD", "RUB":
	default:
		return nil, ErrInvalidSecondaryProjectCurrency
	}
	if normalized == primary {
		return nil, ErrSecondaryCurrencyMustDiffer
	}
	return &normalized, nil
}

func normalizeProjectCategory(category string) (string, error) {
	normalized := canonicalProjectCategory(category)
	if len(normalized) < 2 || len(normalized) > 80 {
		return "", ErrInvalidProjectCategory
	}
	return normalized, nil
}

func canonicalProjectCategory(category string) string {
	normalized := strings.TrimSpace(category)
	if normalized == "" {
		return "Общее"
	}
	return normalized
}
