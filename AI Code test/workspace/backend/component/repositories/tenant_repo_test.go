package repositories_test

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"claimconfig/component/models"
	"claimconfig/component/repositories"
	"claimconfig/server"
)

func openTestDB(t *testing.T) *repositories.TenantRepo {
	t.Helper()
	url := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "repo.db")) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := repositories.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := server.Migrate(context.Background(), db, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	return repositories.NewTenantRepo(db)
}

const cfgJSON = `{"branding":{"company_name":"Acme"}}`

func TestAppendVersion_StaleBaseIsConflict(t *testing.T) {
	ctx := context.Background()
	repo := openTestDB(t)
	now := "2026-10-17T00:00:00.000Z"
	err := repo.Create(ctx, models.Tenant{ID: "t1", Slug: "acme", Name: "Acme", CreatedAt: now, UpdatedAt: now}, cfgJSON, "")
	if err != nil {
		t.Fatal(err)
	}
	in := repositories.AppendVersionInput{TenantID: "t1", BaseVersion: 1, Name: "Acme", ConfigJSON: cfgJSON, Source: "update", Now: now}
	if err := repo.AppendVersion(ctx, in); err != nil {
		t.Fatal(err)
	}
	// A second editor who also started from v1 must not overwrite v2.
	if err := repo.AppendVersion(ctx, in); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	d, err := repo.Get(ctx, "acme")
	if err != nil || d.CurrentVersion != 2 {
		t.Fatalf("current = %+v, %v", d, err)
	}
	if err := repo.Create(ctx, models.Tenant{ID: "t2", Slug: "acme", Name: "X", CreatedAt: now, UpdatedAt: now}, cfgJSON, ""); !errors.Is(err, repositories.ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	if missing, err := repo.Get(ctx, "nope"); missing != nil || err != nil {
		t.Fatalf("missing tenant = %v, %v", missing, err)
	}
}
