package auth

type Permission string

const (
	PermProjectRead   Permission = "project:read"
	PermProjectWrite  Permission = "project:write"
	PermMembersManage Permission = "members:manage"
	PermFinanceRead   Permission = "finance:read"
	PermFinanceWrite  Permission = "finance:write"
	PermDashboardRead Permission = "dashboard:read"
	PermForecastRead  Permission = "forecast:read"
	PermForecastWrite Permission = "forecast:write"
)

var rolePermissions = map[string]map[Permission]bool{
	"PM": {
		PermProjectRead:   true,
		PermProjectWrite:  true,
		PermMembersManage: true,
		PermFinanceRead:   true,
		PermFinanceWrite:  true,
		PermDashboardRead: true,
		PermForecastRead:  true,
		PermForecastWrite: true,
	},
	"ANALYST": {
		PermProjectRead:   true,
		PermFinanceRead:   true,
		PermFinanceWrite:  true,
		PermDashboardRead: true,
		PermForecastRead:  true,
		PermForecastWrite: true,
	},
	"TEAM": {
		PermProjectRead:   true,
		PermFinanceRead:   true,
		PermDashboardRead: true,
		PermForecastRead:  true,
	},
	"CUSTOMER": {
		PermProjectRead:   true,
		PermFinanceRead:   true,
		PermDashboardRead: true,
		PermForecastRead:  true,
	},
}

func IsAllowed(role string, permission Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[permission]
}
