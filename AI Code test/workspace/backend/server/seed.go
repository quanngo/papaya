package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"claimconfig/component/models"
	"claimconfig/component/services"
	"claimconfig/seed"
	"claimconfig/utils"
)

type tenantCounter interface {
	Count(ctx context.Context) (int, error)
}

// SeedTenants inserts the sample tenants through the tenant service (so they
// pass the same validation as UI-created ones) when no tenant exists yet.
func SeedTenants(ctx context.Context, counter tenantCounter, tenants services.TenantService) error {
	n, err := counter.Count(ctx)
	if err != nil || n > 0 {
		return err
	}
	list, err := seed.Load()
	if err != nil {
		return err
	}
	for _, s := range list {
		var cfg models.TenantConfig
		dec := json.NewDecoder(bytes.NewReader(s.Config))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			return fmt.Errorf("seed %s: %w", s.Slug, err)
		}
		if _, err := tenants.Create(ctx, models.CreateTenantRequest{Slug: s.Slug, Config: &cfg, Note: "Seed data"}, nil); err != nil {
			return fmt.Errorf("seed %s: %w", s.Slug, err)
		}
		utils.ShowInfoLogs("seeded tenant %s", s.Slug)
	}
	return nil
}
