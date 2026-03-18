package sqlc

import "fmt"

// Compatibility aliases for service code that uses stable domain names.
type ProjectMemberWithUser = ListProjectMembersRow
type FinanceSummary = GetFinanceSummaryByProjectRow

func ValidateRole(role string) error {
	switch ProjectRole(role) {
	case ProjectRolePM, ProjectRoleANALYST, ProjectRoleTEAM, ProjectRoleCUSTOMER:
		return nil
	default:
		return fmt.Errorf("invalid role: %s", role)
	}
}
