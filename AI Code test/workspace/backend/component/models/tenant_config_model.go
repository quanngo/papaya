package models

// Enumerations of the tenant configuration. Adding a value here is the only
// code change a new claim type, event or channel needs.
var (
	ClaimTypes         = []string{"OUTPATIENT", "INPATIENT", "DENTAL", "MATERNITY", "OPTICAL"}
	NotificationEvents = []string{"claim_submitted", "approved", "rejected", "payment_sent"}
	Channels           = []string{"email", "sms", "webhook"}
	CustomFieldTypes   = []string{"text", "number", "select"}
)

// TenantConfig is one tenant's complete, versioned configuration.
// Each top-level section is owned by one module in component/tenantconfig.
type TenantConfig struct {
	Branding      BrandingConfig      `json:"branding"`
	ClaimTypes    ClaimTypesConfig    `json:"claim_types"`
	Approval      ApprovalConfig      `json:"approval"`
	Notifications NotificationsConfig `json:"notifications"`
	SLA           SLAConfig           `json:"sla"`
	CustomFields  []CustomField       `json:"custom_fields"`
}

type BrandingConfig struct {
	CompanyName    string `json:"company_name" example:"CareLink Insurance"`
	LogoURL        string `json:"logo_url" example:"https://placehold.co/160x48?text=CareLink"`
	PrimaryColor   string `json:"primary_color" example:"#6A1B9A"`
	SecondaryColor string `json:"secondary_color" example:"#FFC107"`
}

type ClaimTypesConfig struct {
	Enabled   []string                 `json:"enabled" example:"OUTPATIENT,OPTICAL"`
	Documents map[string]DocumentRules `json:"documents"`
}

type DocumentRules struct {
	Required []string `json:"required" example:"invoice,prescription"`
	Optional []string `json:"optional" example:"lab_results"`
}

type ApprovalConfig struct {
	AutoApprovalThreshold float64 `json:"auto_approval_threshold" example:"10000"`
	Tiers                 []Tier  `json:"tiers"`
}

// Tier routes amounts in [MinAmount, MaxAmount) to Role; MaxAmount nil = no upper bound.
type Tier struct {
	MinAmount float64  `json:"min_amount" example:"10000"`
	MaxAmount *float64 `json:"max_amount" example:"100000" extensions:"x-nullable"`
	Role      string   `json:"role" example:"assessor"`
}

type NotificationsConfig struct {
	Events     map[string]EventRule `json:"events"`
	WebhookURL string               `json:"webhook_url,omitempty" example:"https://hooks.example.com/claims"`
}

type EventRule struct {
	Enabled  bool     `json:"enabled" example:"true"`
	Channels []string `json:"channels" example:"email,webhook"`
	Template string   `json:"template,omitempty" example:""` // empty = default template
}

type SLAConfig struct {
	Timezone    string           `json:"timezone" example:"Asia/Ho_Chi_Minh"`
	Targets     map[string]int   `json:"targets"` // claim type -> business days
	DefaultDays int              `json:"default_days,omitempty" example:"3"`
	Escalation  []EscalationRule `json:"escalation"`
}

type EscalationRule struct {
	AfterBreachDays int      `json:"after_breach_days" example:"1"`
	Notify          []string `json:"notify" example:"team_lead"`
}

type CustomField struct {
	Key      string   `json:"key" example:"policy_number"`
	Label    string   `json:"label" example:"Policy Number"`
	Type     string   `json:"type" example:"text" enums:"text,number,select"`
	Required bool     `json:"required" example:"true"`
	Pattern  string   `json:"pattern,omitempty" example:"^CL-[0-9]{6}$"`
	Options  []string `json:"options,omitempty"`
}

// Issue is one validation problem. Code is stable; Text is filled per request
// language by the handler (issue.<CODE> in i18n/locales).
type Issue struct {
	Path   string         `json:"path" example:"approval.tiers[1].min_amount"`
	Code   string         `json:"code" example:"TIER_GAP"`
	Params map[string]any `json:"params,omitempty" swaggertype:"object"`
	Text   string         `json:"text" example:"Tier must start at 100000 (where the previous tier ends)."`
}
