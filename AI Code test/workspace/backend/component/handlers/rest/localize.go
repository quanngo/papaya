package rest

import (
	"claimconfig/component/models"
	"claimconfig/i18n"

	"github.com/gin-gonic/gin"
)

// Codes never change with the language; only the Text fields are filled here.

func localizeIssues(c *gin.Context, issues []models.Issue) []models.Issue {
	loc := i18n.Locale(c)
	out := make([]models.Issue, len(issues))
	for i, is := range issues {
		is.Text = i18n.T(loc, "issue."+is.Code, is.Params)
		out[i] = is
	}
	return out
}

func localizeResult(c *gin.Context, r *models.ProcessResult) {
	loc := i18n.Locale(c)
	r.Errors = localizeIssues(c, r.Errors)
	for i := range r.CustomFields {
		f := &r.CustomFields[i]
		if f.Code != "" {
			f.Text = i18n.T(loc, "issue."+f.Code, map[string]any{"field": f.Label})
		}
	}
	for i := range r.Trace {
		r.Trace[i].Text = i18n.T(loc, "trace."+r.Trace[i].Code, r.Trace[i].Params)
	}
}

// localizeUser returns a copy with role and permission names in the request language.
func localizeUser(c *gin.Context, u *models.SessionUser) models.SessionUser {
	loc := i18n.Locale(c)
	out := *u
	if name := i18n.T(loc, "role."+u.Role.Code, nil); name != "" {
		out.Role.Name = name
	}
	out.PermissionNames = make(map[string]string, len(u.Permissions))
	for _, p := range u.Permissions {
		out.PermissionNames[p] = i18n.T(loc, "permission."+p, nil)
	}
	return out
}
