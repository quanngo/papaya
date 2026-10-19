package tenantconfig

import (
	"strings"

	"claimconfig/component/models"
)

// defaultTemplates is used for an enabled event without a custom template.
var defaultTemplates = map[string]string{
	"claim_submitted": "Your {{claim_type}} claim of {{amount}} has been received and is being processed.",
	"approved":        "Your {{claim_type}} claim of {{amount}} has been approved.",
	"rejected":        "Your {{claim_type}} claim of {{amount}} has been rejected.",
	"payment_sent":    "Payment of {{amount}} for your {{claim_type}} claim has been sent.",
}

// NotificationsModule owns which events notify on which channels, with an
// optional custom template per event.
type NotificationsModule struct{}

func (NotificationsModule) Key() string { return "notifications" }

func (NotificationsModule) Validate(cfg *models.TenantConfig) []models.Issue {
	n := cfg.Notifications
	var out []models.Issue
	usesWebhook := false
	for _, e := range sortedKeys(n.Events) {
		rule := n.Events[e]
		path := "notifications.events." + e
		if !contains(models.NotificationEvents, e) {
			out = append(out, issue(path, "EVENT_UNKNOWN", map[string]any{"event": e}))
			continue
		}
		if rule.Enabled && len(rule.Channels) == 0 {
			out = append(out, issue(path+".channels", "EVENT_NO_CHANNEL", map[string]any{"event": e}))
		}
		for i, ch := range rule.Channels {
			if !contains(models.Channels, ch) {
				out = append(out, issue(idx(path+".channels", i), "CHANNEL_UNKNOWN", map[string]any{"channel": ch}))
			}
			if ch == "webhook" && rule.Enabled {
				usesWebhook = true
			}
		}
	}
	switch {
	case usesWebhook && n.WebhookURL == "":
		out = append(out, issue("notifications.webhook_url", "WEBHOOK_URL_REQUIRED", nil))
	case n.WebhookURL != "" && !validHTTPURL(n.WebhookURL):
		out = append(out, issue("notifications.webhook_url", "URL_INVALID", nil))
	}
	return out
}

func (NotificationsModule) Process(cfg *models.TenantConfig, _ *models.Claim, _ *Ctx, res *models.ProcessResult) {
	if !res.Accepted {
		return
	}
	events := []string{"claim_submitted"}
	if res.Approval.AutoApproved {
		events = append(events, "approved")
	}
	for _, e := range events {
		rule, ok := cfg.Notifications.Events[e]
		if !ok || !rule.Enabled || len(rule.Channels) == 0 {
			trace(res, "notifications", "NOTIFICATION_DISABLED", map[string]any{"event": e})
			continue
		}
		tpl, isDefault := rule.Template, false
		if strings.TrimSpace(tpl) == "" {
			tpl, isDefault = defaultTemplates[e], true
		}
		res.Notifications = append(res.Notifications, models.NotificationResult{
			Event:           e,
			Channels:        append([]string{}, rule.Channels...),
			Template:        tpl,
			DefaultTemplate: isDefault,
		})
		trace(res, "notifications", "NOTIFICATION_SENT", map[string]any{"event": e, "channels": strings.Join(rule.Channels, ", ")})
	}
}

func (NotificationsModule) DiffKeys() map[string]string { return nil }
