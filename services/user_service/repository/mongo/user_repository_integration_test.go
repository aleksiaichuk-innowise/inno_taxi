//go:build integration

package mongo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.mongodb.org/mongo-driver/v2/bson"

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/config"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	mongomigration "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/migrations/mongo"
	repomongo "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

var testRepo *repomongo.UserRepository

func TestMain(m *testing.M) {
	// defers only run on a normal function return, never past os.Exit — so the
	// container/client cleanup must happen inside run(), not in TestMain itself.
	os.Exit(run(m))
}

func run(m *testing.M) int {
	pool, err := dockertest.NewPool("")
	if err != nil {
		fmt.Println("could not connect to docker:", err)
		return 1
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "7",
		Env: []string{
			"MONGO_INITDB_ROOT_USERNAME=taxi",
			"MONGO_INITDB_ROOT_PASSWORD=taxi",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	})
	if err != nil {
		fmt.Println("could not start mongo container:", err)
		return 1
	}
	defer func() { _ = pool.Purge(resource) }()

	cfg := &config.MongoConfig{
		Host:     "localhost",
		Port:     resource.GetPort("27017/tcp"),
		Database: "test",
		Username: "taxi",
		Password: "taxi",
	}

	ctx := context.Background()
	var client *dbmongo.MongoClient
	pool.MaxWait = 60 * time.Second
	if err := pool.Retry(func() error {
		var err error
		client, err = dbmongo.New(ctx, cfg)
		return err
	}); err != nil {
		fmt.Println("could not connect to mongo:", err)
		return 1
	}
	defer func() { _ = client.Close(ctx) }()

	if err := mongomigration.RunMigrations(ctx, *client); err != nil {
		fmt.Println("could not run migrations:", err)
		return 1
	}

	testRepo = repomongo.NewUserRepository(client)

	return m.Run()
}

var idCounter int64

func uniqueID() int64 {
	return atomic.AddInt64(&idCounter, 1)
}

func newTestUser(t *testing.T) *serviceEntity.User {
	t.Helper()
	n := uniqueID()
	now := time.Now()
	return &serviceEntity.User{
		Name:         fmt.Sprintf("Test User %d", n),
		Email:        fmt.Sprintf("test-user-%d@example.com", n),
		Phone:        fmt.Sprintf("+155500%05d", n),
		PasswordHash: "initial-hash",
		Roles:        []serviceEntity.Role{serviceEntity.RoleUser},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func containsRole(roles []serviceEntity.Role, role string) bool {
	for _, r := range roles {
		if string(r) == role {
			return true
		}
	}
	return false
}

func TestUserRepository_CreateUser_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)

	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Error("expected created user to have a non-empty ID")
	}
	if created.Email != u.Email {
		t.Errorf("got email %q, want %q", created.Email, u.Email)
	}
}

func TestUserRepository_CreateUser_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	u1 := newTestUser(t)
	if _, err := testRepo.CreateUser(ctx, u1); err != nil {
		t.Fatalf("setup: create first user: %v", err)
	}

	u2 := newTestUser(t)
	u2.Email = u1.Email

	_, err := testRepo.CreateUser(ctx, u2)
	if !errors.Is(err, errorsx.ErrUserAlreadyExists) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserAlreadyExists)
	}
}

func TestUserRepository_CreateUser_DuplicatePhone(t *testing.T) {
	ctx := context.Background()
	u1 := newTestUser(t)
	if _, err := testRepo.CreateUser(ctx, u1); err != nil {
		t.Fatalf("setup: create first user: %v", err)
	}

	u2 := newTestUser(t)
	u2.Phone = u1.Phone

	_, err := testRepo.CreateUser(ctx, u2)
	if !errors.Is(err, errorsx.ErrUserAlreadyExists) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserAlreadyExists)
	}
}

func TestUserRepository_FindByLogin_ByEmail(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	found, err := testRepo.FindByLogin(ctx, u.Email)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("got ID %q, want %q", found.ID, created.ID)
	}
}

func TestUserRepository_FindByLogin_ByPhone(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	found, err := testRepo.FindByLogin(ctx, u.Phone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("got ID %q, want %q", found.ID, created.ID)
	}
}

func TestUserRepository_FindByLogin_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := testRepo.FindByLogin(ctx, "nonexistent-login@example.com")
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_FindByID_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	found, err := testRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Email != u.Email {
		t.Errorf("got email %q, want %q", found.Email, u.Email)
	}
}

func TestUserRepository_FindByID_InvalidHex(t *testing.T) {
	ctx := context.Background()
	_, err := testRepo.FindByID(ctx, "not-a-valid-object-id")
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_FindByID_ExcludesSoftDeleted(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}
	if err := testRepo.DeleteProfile(ctx, created.ID); err != nil {
		t.Fatalf("setup: delete profile: %v", err)
	}

	_, err = testRepo.FindByID(ctx, created.ID)
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_UpdateProfile_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	n := uniqueID()
	newName := fmt.Sprintf("Updated Name %d", n)
	newEmail := fmt.Sprintf("updated-%d@example.com", n)
	newPhone := fmt.Sprintf("+155599%05d", n)

	updated, err := testRepo.UpdateProfile(ctx, created.ID, newName, newEmail, newPhone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName || updated.Email != newEmail || updated.Phone != newPhone {
		t.Errorf("got %+v, want name=%q email=%q phone=%q", updated, newName, newEmail, newPhone)
	}
}

func TestUserRepository_UpdateProfile_DuplicateEmail(t *testing.T) {
	ctx := context.Background()
	u1 := newTestUser(t)
	if _, err := testRepo.CreateUser(ctx, u1); err != nil {
		t.Fatalf("setup: create first user: %v", err)
	}
	u2 := newTestUser(t)
	created2, err := testRepo.CreateUser(ctx, u2)
	if err != nil {
		t.Fatalf("setup: create second user: %v", err)
	}

	_, err = testRepo.UpdateProfile(ctx, created2.ID, u2.Name, u1.Email, u2.Phone)
	if !errors.Is(err, errorsx.ErrUserAlreadyExists) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserAlreadyExists)
	}
}

func TestUserRepository_UpdateProfile_NotFound(t *testing.T) {
	ctx := context.Background()
	n := uniqueID()
	_, err := testRepo.UpdateProfile(ctx, bson.NewObjectID().Hex(), "x", fmt.Sprintf("missing-%d@example.com", n), "+15550000000")
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_SetPassword_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	const newHash = "new-hashed-password"
	if err := testRepo.SetPassword(ctx, created.ID, newHash); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, err := testRepo.FindByLogin(ctx, u.Email)
	if err != nil {
		t.Fatalf("find after set password: %v", err)
	}
	if found.PasswordHash != newHash {
		t.Errorf("got password hash %q, want %q", found.PasswordHash, newHash)
	}
}

func TestUserRepository_SetPassword_NotFound(t *testing.T) {
	ctx := context.Background()
	err := testRepo.SetPassword(ctx, bson.NewObjectID().Hex(), "irrelevant")
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_DeleteProfile_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	if err := testRepo.DeleteProfile(ctx, created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = testRepo.FindByID(ctx, created.ID)
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v after delete, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_DeleteProfile_Idempotent(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	if err := testRepo.DeleteProfile(ctx, created.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	err = testRepo.DeleteProfile(ctx, created.ID)
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v on second delete, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserRepository_AddRole_Success(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	if err := testRepo.AddRole(ctx, created.ID, "analyst"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, err := testRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("find after add role: %v", err)
	}
	if !containsRole(found.Roles, "analyst") || !containsRole(found.Roles, "user") {
		t.Errorf("got roles %v, want to contain both user and analyst", found.Roles)
	}
}

func TestUserRepository_AddRole_Dedup(t *testing.T) {
	ctx := context.Background()
	u := newTestUser(t)
	created, err := testRepo.CreateUser(ctx, u)
	if err != nil {
		t.Fatalf("setup: create user: %v", err)
	}

	if err := testRepo.AddRole(ctx, created.ID, "analyst"); err != nil {
		t.Fatalf("first add role: %v", err)
	}
	if err := testRepo.AddRole(ctx, created.ID, "analyst"); err != nil {
		t.Fatalf("second add role: %v", err)
	}

	found, err := testRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("find after add role: %v", err)
	}
	count := 0
	for _, r := range found.Roles {
		if r == "analyst" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("got %d 'analyst' entries in roles %v, want 1 (deduped)", count, found.Roles)
	}
}

func TestUserRepository_AddRole_NotFound(t *testing.T) {
	ctx := context.Background()
	err := testRepo.AddRole(ctx, bson.NewObjectID().Hex(), "analyst")
	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}
