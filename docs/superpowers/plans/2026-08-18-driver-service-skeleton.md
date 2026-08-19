# Driver Service Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Note for this project:** this plan is executed by the human, not by Claude — Claude's role here is task-setting and review only (see project convention). Use the task breakdown as your own checklist; ask for review after each task rather than having Claude execute it.

**Goal:** Stand up `services/driver_service` as a second microservice with the same Clean Architecture shape as `user_service`: driver profile view/update and availability-status toggling over HTTP, JWT-protected, with unit + Mongo-integration tests, graceful shutdown, and Docker Compose wiring.

**Architecture:** Mirrors `user_service` exactly — `handler/http` → `service` → `repository/mongo`, domain entities in `entity/service` (tag-free), DB projections in `entity/repository` (bson-tagged), a local `errorsx` package for domain errors (not `shared/errorsx` — see Global Constraints), JWT auth middleware copied from `user_service` (services never import each other, per `ARCHITECTURE.md`).

**Tech Stack:** Go 1.26.1, Gin, MongoDB (`go.mongodb.org/mongo-driver/v2`), `go-playground/validator/v10`, `golang-jwt/jwt/v5`, `ory/dockertest/v3` for integration tests.

**Spec:** `docs/superpowers/specs/2026-08-18-driver-service-design.md`

## Global Constraints

- Go version: `1.26.1` (match `user_service/go.mod`).
- Module path: `github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service`.
- `shared` module dependency via `replace github.com/aleksiaichuk-innowise/inno_taxi/shared => ../../shared` (same as `user_service/go.mod`).
- **Driver-domain errors live in a new local `errorsx` package inside `driver_service`, NOT in `shared/errorsx`.** `ARCHITECTURE.md`'s Cons section (#7) flags mixing domain errors from different services in one shared package as a problem — `shared/errorsx` already holds `ErrUserNotFound`/`ErrPaymentNotFound` together, which the doc calls out as coupling that shouldn't be repeated. Keep using `shared/errorsx.HttpErrResp` and `shared/errorsx.ErrInternal` (transport-neutral, not domain-specific), but define `ErrDriverNotFound`, `ErrDriverAlreadyExists`, `ErrInvalidTaxiType`, `ErrInvalidStatus` locally.
- JWT contract: HS256, shared secret via `JWT_SECRET` env var, claims `sub` (user id) + `roles []string` — identical contract already used by `user_service`'s `middleware.Auth`. Copy that middleware file verbatim (adjusted import paths); do not attempt to share it as a package between services.
- HTTP port default: `8081` (host mapping in Docker Compose) — `8080` is already `user_service`'s.
- Mongo database name: `driver_taxi` (distinct from `user_service`'s `taxi`, same Mongo server is fine — Mongo supports multiple databases per connection).
- Self-service endpoints (`/profile`, `/profile/status`) take identity from the JWT (`c.GetString("userID")`), never from a path parameter — same rule already enforced in `user_service`.
- Integration tests (`repository/mongo`) use a build tag `integration` and `ory/dockertest/v3`, exactly like `services/user_service/repository/mongo/user_repository_integration_test.go`. **`TestMain` must return the exit code via a `run(m) int` helper function and call `os.Exit(run(m))` from `TestMain` — never `os.Exit(m.Run())` directly with `defer` cleanup in the same function.** `os.Exit` does not run deferred functions; this exact bug (leaked Mongo container) was caught and fixed in `user_service`'s integration tests — don't reintroduce it.

---

### Task 1: Module scaffold, config, domain entities

**Files:**
- Create: `services/driver_service/go.mod`
- Create: `services/driver_service/config/config.go`
- Create: `services/driver_service/entity/service/driver.go`
- Create: `services/driver_service/errorsx/errors.go`
- Create: `services/driver_service/app/db/mongo/mongo.go`

**Interfaces:**
- Produces: `service.TaxiType` (`TaxiTypeEconomy`/`TaxiTypeComfort`/`TaxiTypeBusiness`, method `IsValid() bool`), `service.Status` (`StatusAvailable`/`StatusOnTrip`/`StatusOffline`, method `IsValid() bool`), `service.Driver{ID, UserID, TaxiType, Status, CreatedAt, UpdatedAt}`, `service.CreateDriverInput{UserID, TaxiType}`. `errorsx.ErrDriverNotFound`, `errorsx.ErrDriverAlreadyExists`, `errorsx.ErrInvalidTaxiType`, `errorsx.ErrInvalidStatus`. `config.Config{Mongo, Host, JWT}`, `config.MongoConfig{Host, Port, Database, Username, Password}`, `config.HostConfig{Host, Port}`, `config.JWTConfig{Secret}`, `config.Load() *Config`. `dbmongo.MongoClient{Client, Database}`, `dbmongo.New(ctx, cfg *config.MongoConfig) (*MongoClient, error)`, `(*MongoClient) Close(ctx) error`.

- [ ] **Step 1: Create the Go module**

Run:
```bash
mkdir -p services/driver_service
cd services/driver_service
go mod init github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service
```

Then edit `go.mod` to match the module's Go version and add the `shared` replace directive:

```go
module github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service

go 1.26.1

require github.com/aleksiaichuk-innowise/inno_taxi/shared v0.0.0-00010101000000-000000000000

replace github.com/aleksiaichuk-innowise/inno_taxi/shared => ../../shared
```

- [ ] **Step 2: Add domain entities**

Create `entity/service/driver.go`:

```go
package service

import "time"

type TaxiType string

const (
	TaxiTypeEconomy  TaxiType = "economy"
	TaxiTypeComfort  TaxiType = "comfort"
	TaxiTypeBusiness TaxiType = "business"
)

func (t TaxiType) IsValid() bool {
	switch t {
	case TaxiTypeEconomy, TaxiTypeComfort, TaxiTypeBusiness:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusAvailable Status = "available"
	StatusOnTrip    Status = "on-trip"
	StatusOffline   Status = "offline"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusAvailable, StatusOnTrip, StatusOffline:
		return true
	default:
		return false
	}
}

type Driver struct {
	ID        string
	UserID    string
	TaxiType  TaxiType
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateDriverInput struct {
	UserID   string
	TaxiType TaxiType
}
```

- [ ] **Step 3: Add local domain errors**

Create `errorsx/errors.go`:

```go
package errorsx

import "errors"

var (
	ErrDriverNotFound      = errors.New("driver not found")
	ErrDriverAlreadyExists = errors.New("driver already exists")
	ErrInvalidTaxiType     = errors.New("invalid taxi type")
	ErrInvalidStatus       = errors.New("invalid status")
)
```

- [ ] **Step 4: Add config**

Create `config/config.go`:

```go
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Mongo *MongoConfig
	Host  *HostConfig
	JWT   *JWTConfig
}

type MongoConfig struct {
	Host     string `env:"MONGO_HOST"`
	Port     string `env:"MONGO_PORT"`
	Database string `env:"MONGO_DB"`
	Username string `env:"MONGO_USERNAME"`
	Password string `env:"MONGO_PASSWORD"`
}

type HostConfig struct {
	Host string `env:"HTTP_HOST"`
	Port string `env:"HTTP_PORT"`
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET"`
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Mongo: &MongoConfig{
			Host:     getEnv("MONGO_HOST", "localhost"),
			Port:     getEnv("MONGO_PORT", "27017"),
			Database: getEnv("MONGO_DB", "driver_taxi"),
			Username: getEnv("MONGO_USERNAME", "taxi"),
			Password: getEnv("MONGO_PASSWORD", "taxi"),
		},
		Host: &HostConfig{
			Host: getEnv("HTTP_HOST", "localhost"),
			Port: getEnv("HTTP_PORT", "8081"),
		},
		JWT: &JWTConfig{
			Secret: getEnv("JWT_SECRET", "taxi"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 5: Add the Mongo client wrapper**

Create `app/db/mongo/mongo.go` (identical shape to `services/user_service/app/db/mongo/mongo.go`, just the config import path changes):

```go
package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func New(ctx context.Context, cfg *config.MongoConfig) (*MongoClient, error) {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port,
	)

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}

	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}

	return &MongoClient{
		Client:   client,
		Database: client.Database(cfg.Database),
	}, nil
}

func (m *MongoClient) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
```

- [ ] **Step 6: Add remaining direct dependencies and verify it builds**

Run:
```bash
cd services/driver_service
go get github.com/joho/godotenv@v1.5.1
go get go.mongodb.org/mongo-driver/v2@v2.8.0
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 7: Commit**

```bash
git add services/driver_service/go.mod services/driver_service/go.sum \
  services/driver_service/config services/driver_service/entity \
  services/driver_service/errorsx services/driver_service/app
git commit -m "feat(driver_service): scaffold module, config, and domain entities"
```

---

### Task 2: Repository layer (Mongo)

**Files:**
- Create: `services/driver_service/entity/repository/driver.go`
- Create: `services/driver_service/migrations/mongo/migration.go`
- Create: `services/driver_service/migrations/mongo/runner.go`
- Create: `services/driver_service/repository/mongo/driver_repository.go`
- Create: `services/driver_service/repository/mongo/create_driver.go`
- Create: `services/driver_service/repository/mongo/find_by_user_id.go`
- Create: `services/driver_service/repository/mongo/update_profile.go`
- Create: `services/driver_service/repository/mongo/update_status.go`
- Test: `services/driver_service/repository/mongo/driver_repository_integration_test.go`

**Interfaces:**
- Consumes: `service.TaxiType`, `service.Status`, `service.Driver`, `service.CreateDriverInput` (Task 1), `errorsx.ErrDriverNotFound`, `errorsx.ErrDriverAlreadyExists` (Task 1), `dbmongo.MongoClient`, `dbmongo.New` (Task 1).
- Produces: `mongo.DriverRepository` struct with methods `CreateDriver(ctx, *serviceEntity.Driver) (*serviceEntity.Driver, error)`, `FindByUserID(ctx, userID string) (*serviceEntity.Driver, error)`, `UpdateProfile(ctx, userID, taxiType string) (*serviceEntity.Driver, error)`, `UpdateStatus(ctx, userID, status string) (*serviceEntity.Driver, error)`; `mongo.NewDriverRepository(client *dbmongo.MongoClient) *DriverRepository`.

- [ ] **Step 1: Add the DB projection entity**

Create `entity/repository/driver.go`:

```go
package repository

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Driver struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    string        `bson:"user_id"`
	TaxiType  string        `bson:"taxi_type"`
	Status    string        `bson:"status"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}
```

- [ ] **Step 2: Add the migration (unique index on `user_id`)**

Create `migrations/mongo/migration.go`:

```go
package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, db *mongo.Database) error
}

var All = []Migration{
	{
		Version: 1,
		Name:    "unique_index_user_id",
		Up: func(ctx context.Context, db *mongo.Database) error {
			_, err := db.Collection("drivers").Indexes().CreateOne(ctx, mongo.IndexModel{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("uniq_user_id"),
			})
			return err
		},
	},
}
```

Create `migrations/mongo/runner.go` (identical shape to `services/user_service/migrations/mongo/runner.go`):

```go
package mongo

import (
	"context"
	"errors"
	"fmt"

	client "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type appliedMigration struct {
	Version int    `bson:"version"`
	Name    string `bson:"name"`
}

func RunMigrations(ctx context.Context, c client.MongoClient) error {
	col := c.Database.Collection("migrations")

	opts := options.FindOne().SetSort(bson.M{"version": -1})

	var last appliedMigration
	err := col.FindOne(ctx, bson.M{"version": 1}, opts).Decode(&last)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("find migrations error: %w", err)
	}
	for _, m := range All {
		if last.Version < m.Version {
			if err := m.Up(ctx, c.Database); err != nil {
				return fmt.Errorf("up migration error: %w", err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 3: Add the repository aggregate**

Create `repository/mongo/driver_repository.go`:

```go
package mongo

import (
	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
)

const driversCollection = "drivers"

type DriverRepository struct {
	client *dbmongo.MongoClient
}

func NewDriverRepository(client *dbmongo.MongoClient) *DriverRepository {
	return &DriverRepository{client: client}
}
```

(The `var _ service.DriverRepository = (*DriverRepository)(nil)` compile-time
assertion is added in Task 3, once the `service` package's interface exists —
adding it now would create an import cycle since `service` doesn't exist yet.)

- [ ] **Step 4: Implement `CreateDriver`**

Create `repository/mongo/create_driver.go`:

```go
package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (r DriverRepository) CreateDriver(ctx context.Context, driver *serviceEntity.Driver) (*serviceEntity.Driver, error) {
	doc := repoEntity.Driver{
		ID:        bson.NewObjectID(),
		UserID:    driver.UserID,
		TaxiType:  string(driver.TaxiType),
		Status:    string(driver.Status),
		CreatedAt: driver.CreatedAt,
		UpdatedAt: driver.UpdatedAt,
	}

	if _, err := r.client.Database.Collection(driversCollection).InsertOne(ctx, doc); err != nil {
		if drivermongo.IsDuplicateKeyError(err) {
			return nil, errorsx.ErrDriverAlreadyExists
		}
		return nil, err
	}

	created := *driver
	created.ID = doc.ID.Hex()
	return &created, nil
}
```

- [ ] **Step 5: Implement `FindByUserID`**

Create `repository/mongo/find_by_user_id.go`:

```go
package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (r DriverRepository) FindByUserID(ctx context.Context, userID string) (*serviceEntity.Driver, error) {
	filter := bson.M{"user_id": userID}
	res := r.client.Database.Collection(driversCollection).FindOne(ctx, filter)

	var doc repoEntity.Driver
	if err := res.Decode(&doc); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return nil, errorsx.ErrDriverNotFound
		}
		return nil, err
	}

	return &serviceEntity.Driver{
		ID:        doc.ID.Hex(),
		UserID:    doc.UserID,
		TaxiType:  serviceEntity.TaxiType(doc.TaxiType),
		Status:    serviceEntity.Status(doc.Status),
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}
```

- [ ] **Step 6: Implement `UpdateProfile`**

Create `repository/mongo/update_profile.go`:

```go
package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (r DriverRepository) UpdateProfile(ctx context.Context, userID string, taxiType string) (*serviceEntity.Driver, error) {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{
		"taxi_type":  taxiType,
		"updated_at": time.Now(),
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	res := r.client.Database.Collection(driversCollection).FindOneAndUpdate(ctx, filter, update, opts)

	var doc repoEntity.Driver
	if err := res.Decode(&doc); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return nil, errorsx.ErrDriverNotFound
		}
		return nil, err
	}

	return &serviceEntity.Driver{
		ID:        doc.ID.Hex(),
		UserID:    doc.UserID,
		TaxiType:  serviceEntity.TaxiType(doc.TaxiType),
		Status:    serviceEntity.Status(doc.Status),
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}
```

- [ ] **Step 7: Implement `UpdateStatus`**

Create `repository/mongo/update_status.go` (same shape as `update_profile.go`, `status` field instead of `taxi_type`):

```go
package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	repoEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/repository"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (r DriverRepository) UpdateStatus(ctx context.Context, userID string, status string) (*serviceEntity.Driver, error) {
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{
		"status":     status,
		"updated_at": time.Now(),
	}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	res := r.client.Database.Collection(driversCollection).FindOneAndUpdate(ctx, filter, update, opts)

	var doc repoEntity.Driver
	if err := res.Decode(&doc); err != nil {
		if errors.Is(err, drivermongo.ErrNoDocuments) {
			return nil, errorsx.ErrDriverNotFound
		}
		return nil, err
	}

	return &serviceEntity.Driver{
		ID:        doc.ID.Hex(),
		UserID:    doc.UserID,
		TaxiType:  serviceEntity.TaxiType(doc.TaxiType),
		Status:    serviceEntity.Status(doc.Status),
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}, nil
}
```

- [ ] **Step 8: Add dependencies, verify it builds**

```bash
cd services/driver_service
go build ./...
```
Expected: exits 0.

- [ ] **Step 9: Write the integration test**

Create `repository/mongo/driver_repository_integration_test.go`:

```go
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

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	mongomigration "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/migrations/mongo"
	repomongo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/repository/mongo"
)

var testRepo *repomongo.DriverRepository

func TestMain(m *testing.M) {
	// os.Exit skips deferred functions — cleanup must happen inside run(),
	// never in TestMain itself.
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

	testRepo = repomongo.NewDriverRepository(client)

	return m.Run()
}

var idCounter int64

func uniqueID() int64 {
	return atomic.AddInt64(&idCounter, 1)
}

func newTestDriver(t *testing.T) *serviceEntity.Driver {
	t.Helper()
	n := uniqueID()
	now := time.Now()
	return &serviceEntity.Driver{
		UserID:    fmt.Sprintf("user-%d", n),
		TaxiType:  serviceEntity.TaxiTypeEconomy,
		Status:    serviceEntity.StatusOffline,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestDriverRepository_CreateDriver_Success(t *testing.T) {
	ctx := context.Background()
	d := newTestDriver(t)

	created, err := testRepo.CreateDriver(ctx, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Error("expected created driver to have a non-empty ID")
	}
	if created.UserID != d.UserID {
		t.Errorf("got user id %q, want %q", created.UserID, d.UserID)
	}
}

func TestDriverRepository_CreateDriver_DuplicateUserID(t *testing.T) {
	ctx := context.Background()
	d1 := newTestDriver(t)
	if _, err := testRepo.CreateDriver(ctx, d1); err != nil {
		t.Fatalf("setup: create first driver: %v", err)
	}

	d2 := newTestDriver(t)
	d2.UserID = d1.UserID

	_, err := testRepo.CreateDriver(ctx, d2)
	if !errors.Is(err, errorsx.ErrDriverAlreadyExists) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrDriverAlreadyExists)
	}
}

func TestDriverRepository_FindByUserID_Success(t *testing.T) {
	ctx := context.Background()
	d := newTestDriver(t)
	created, err := testRepo.CreateDriver(ctx, d)
	if err != nil {
		t.Fatalf("setup: create driver: %v", err)
	}

	found, err := testRepo.FindByUserID(ctx, d.UserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("got ID %q, want %q", found.ID, created.ID)
	}
}

func TestDriverRepository_FindByUserID_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := testRepo.FindByUserID(ctx, "nonexistent-user")
	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrDriverNotFound)
	}
}

func TestDriverRepository_UpdateProfile_Success(t *testing.T) {
	ctx := context.Background()
	d := newTestDriver(t)
	if _, err := testRepo.CreateDriver(ctx, d); err != nil {
		t.Fatalf("setup: create driver: %v", err)
	}

	updated, err := testRepo.UpdateProfile(ctx, d.UserID, string(serviceEntity.TaxiTypeBusiness))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.TaxiType != serviceEntity.TaxiTypeBusiness {
		t.Errorf("got taxi type %q, want %q", updated.TaxiType, serviceEntity.TaxiTypeBusiness)
	}
}

func TestDriverRepository_UpdateProfile_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := testRepo.UpdateProfile(ctx, "nonexistent-user", string(serviceEntity.TaxiTypeComfort))
	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrDriverNotFound)
	}
}

func TestDriverRepository_UpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	d := newTestDriver(t)
	if _, err := testRepo.CreateDriver(ctx, d); err != nil {
		t.Fatalf("setup: create driver: %v", err)
	}

	updated, err := testRepo.UpdateStatus(ctx, d.UserID, string(serviceEntity.StatusAvailable))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != serviceEntity.StatusAvailable {
		t.Errorf("got status %q, want %q", updated.Status, serviceEntity.StatusAvailable)
	}
}

func TestDriverRepository_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := testRepo.UpdateStatus(ctx, "nonexistent-user", string(serviceEntity.StatusOnTrip))
	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrDriverNotFound)
	}
}
```

- [ ] **Step 10: Add the dockertest dependency and run the integration tests**

```bash
cd services/driver_service
go get github.com/ory/dockertest/v3@latest
DOCKER_HOST="unix:///home/user/.docker/desktop/docker.sock" go test -tags=integration ./repository/mongo/... -v
```

Expected: all `TestDriverRepository_*` tests PASS. (If Docker's socket is at
the default path on your machine, drop the `DOCKER_HOST` prefix — check with
`docker context inspect` if `docker ps` fails.)

- [ ] **Step 11: Verify the non-integration build/vet is unaffected**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: all exit 0; `repository/mongo` reports `[no test files]` for the
plain `go test ./...` run (the integration test file is excluded by the
build tag).

- [ ] **Step 12: Commit**

```bash
git add services/driver_service/entity/repository services/driver_service/migrations \
  services/driver_service/repository services/driver_service/go.mod services/driver_service/go.sum
git commit -m "feat(driver_service): add Mongo repository layer with integration tests"
```

---

### Task 3: Service layer

**Files:**
- Create: `services/driver_service/service/service.go`
- Create: `services/driver_service/service/create_driver.go`
- Create: `services/driver_service/service/get_profile.go`
- Create: `services/driver_service/service/update_profile.go`
- Create: `services/driver_service/service/update_status.go`
- Modify: `services/driver_service/repository/mongo/driver_repository.go` — add the interface assertion now that `service.DriverRepository` exists
- Test: `services/driver_service/service/driver_service_test.go`

**Interfaces:**
- Consumes: everything from Task 1 and Task 2's method signatures.
- Produces: `service.DriverRepository` interface, `service.DriverService` struct, `service.NewDriverService(repo DriverRepository) *DriverService`, methods `CreateDriver(ctx, serviceEntity.CreateDriverInput) (*serviceEntity.Driver, error)`, `GetProfile(ctx, userID string) (*serviceEntity.Driver, error)`, `UpdateProfile(ctx, userID string, taxiType serviceEntity.TaxiType) (*serviceEntity.Driver, error)`, `UpdateStatus(ctx, userID string, status serviceEntity.Status) (*serviceEntity.Driver, error)`.

- [ ] **Step 1: Write the failing unit tests**

Create `service/driver_service_test.go`:

```go
package service

import (
	"context"
	"errors"
	"testing"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

type fakeDriverRepository struct {
	driver *serviceEntity.Driver
	err    error

	updateProfileDriver *serviceEntity.Driver
	updateProfileErr    error

	updateStatusDriver *serviceEntity.Driver
	updateStatusErr    error

	createErr error
}

func (f *fakeDriverRepository) CreateDriver(_ context.Context, driver *serviceEntity.Driver) (*serviceEntity.Driver, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return driver, nil
}

func (f *fakeDriverRepository) FindByUserID(_ context.Context, _ string) (*serviceEntity.Driver, error) {
	return f.driver, f.err
}

func (f *fakeDriverRepository) UpdateProfile(_ context.Context, _, _ string) (*serviceEntity.Driver, error) {
	return f.updateProfileDriver, f.updateProfileErr
}

func (f *fakeDriverRepository) UpdateStatus(_ context.Context, _, _ string) (*serviceEntity.Driver, error) {
	return f.updateStatusDriver, f.updateStatusErr
}

func TestDriverService_CreateDriver_Success(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)

	got, err := svc.CreateDriver(context.Background(), serviceEntity.CreateDriverInput{
		UserID:   "user-1",
		TaxiType: serviceEntity.TaxiTypeEconomy,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("got user id %q, want %q", got.UserID, "user-1")
	}
	if got.Status != serviceEntity.StatusOffline {
		t.Errorf("got status %q, want default %q", got.Status, serviceEntity.StatusOffline)
	}
}

func TestDriverService_CreateDriver_InvalidTaxiType(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)

	_, err := svc.CreateDriver(context.Background(), serviceEntity.CreateDriverInput{
		UserID:   "user-1",
		TaxiType: serviceEntity.TaxiType("bogus"),
	})

	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidTaxiType)
	}
}

func TestDriverService_GetProfile_Success(t *testing.T) {
	repo := &fakeDriverRepository{driver: &serviceEntity.Driver{UserID: "user-1"}}
	svc := NewDriverService(repo)

	got, err := svc.GetProfile(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("got user id %q, want %q", got.UserID, "user-1")
	}
}

func TestDriverService_GetProfile_NotFound(t *testing.T) {
	repo := &fakeDriverRepository{err: errorsx.ErrDriverNotFound}
	svc := NewDriverService(repo)

	_, err := svc.GetProfile(context.Background(), "missing-user")

	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrDriverNotFound)
	}
}

func TestDriverService_UpdateProfile_Success(t *testing.T) {
	updated := &serviceEntity.Driver{UserID: "user-1", TaxiType: serviceEntity.TaxiTypeBusiness}
	repo := &fakeDriverRepository{updateProfileDriver: updated}
	svc := NewDriverService(repo)

	got, err := svc.UpdateProfile(context.Background(), "user-1", serviceEntity.TaxiTypeBusiness)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != updated {
		t.Errorf("got %+v, want %+v", got, updated)
	}
}

func TestDriverService_UpdateProfile_InvalidTaxiType(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)

	_, err := svc.UpdateProfile(context.Background(), "user-1", serviceEntity.TaxiType("bogus"))

	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidTaxiType)
	}
}

func TestDriverService_UpdateStatus_Success(t *testing.T) {
	updated := &serviceEntity.Driver{UserID: "user-1", Status: serviceEntity.StatusAvailable}
	repo := &fakeDriverRepository{updateStatusDriver: updated}
	svc := NewDriverService(repo)

	got, err := svc.UpdateStatus(context.Background(), "user-1", serviceEntity.StatusAvailable)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != updated {
		t.Errorf("got %+v, want %+v", got, updated)
	}
}

func TestDriverService_UpdateStatus_InvalidStatus(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)

	_, err := svc.UpdateStatus(context.Background(), "user-1", serviceEntity.Status("bogus"))

	if !errors.Is(err, errorsx.ErrInvalidStatus) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidStatus)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/driver_service
go test ./service/... -v
```

Expected: FAIL — compile errors (`NewDriverService`, `DriverRepository`,
etc. undefined). This is the correct RED: the feature doesn't exist yet.

- [ ] **Step 3: Implement the service interface and struct**

Create `service/service.go`:

```go
package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

type DriverRepository interface {
	CreateDriver(ctx context.Context, driver *serviceEntity.Driver) (*serviceEntity.Driver, error)
	FindByUserID(ctx context.Context, userID string) (*serviceEntity.Driver, error)
	UpdateProfile(ctx context.Context, userID, taxiType string) (*serviceEntity.Driver, error)
	UpdateStatus(ctx context.Context, userID, status string) (*serviceEntity.Driver, error)
}

type DriverService struct {
	driverRepo DriverRepository
}

func NewDriverService(driverRepo DriverRepository) *DriverService {
	return &DriverService{driverRepo}
}
```

- [ ] **Step 4: Implement `CreateDriver`**

Create `service/create_driver.go`:

```go
package service

import (
	"context"
	"time"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) CreateDriver(ctx context.Context, input serviceEntity.CreateDriverInput) (*serviceEntity.Driver, error) {
	if !input.TaxiType.IsValid() {
		return nil, errorsx.ErrInvalidTaxiType
	}

	now := time.Now()
	driver := &serviceEntity.Driver{
		UserID:    input.UserID,
		TaxiType:  input.TaxiType,
		Status:    serviceEntity.StatusOffline,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return s.driverRepo.CreateDriver(ctx, driver)
}
```

- [ ] **Step 5: Implement `GetProfile`**

Create `service/get_profile.go`:

```go
package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

func (s DriverService) GetProfile(ctx context.Context, userID string) (*serviceEntity.Driver, error) {
	return s.driverRepo.FindByUserID(ctx, userID)
}
```

- [ ] **Step 6: Implement `UpdateProfile`**

Create `service/update_profile.go`:

```go
package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) UpdateProfile(ctx context.Context, userID string, taxiType serviceEntity.TaxiType) (*serviceEntity.Driver, error) {
	if !taxiType.IsValid() {
		return nil, errorsx.ErrInvalidTaxiType
	}
	return s.driverRepo.UpdateProfile(ctx, userID, string(taxiType))
}
```

- [ ] **Step 7: Implement `UpdateStatus`**

Create `service/update_status.go`:

```go
package service

import (
	"context"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) UpdateStatus(ctx context.Context, userID string, status serviceEntity.Status) (*serviceEntity.Driver, error) {
	if !status.IsValid() {
		return nil, errorsx.ErrInvalidStatus
	}
	return s.driverRepo.UpdateStatus(ctx, userID, string(status))
}
```

- [ ] **Step 8: Run the tests again to verify they pass**

```bash
go test ./service/... -v
```

Expected: all `TestDriverService_*` tests PASS.

- [ ] **Step 9: Add the compile-time interface assertion to the repository**

Modify `repository/mongo/driver_repository.go` — add the `service` import and
the assertion line, matching the pattern already used in
`services/user_service/repository/mongo/user_repository.go`:

```go
package mongo

import (
	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
)

const driversCollection = "drivers"

var _ service.DriverRepository = (*DriverRepository)(nil)

type DriverRepository struct {
	client *dbmongo.MongoClient
}

func NewDriverRepository(client *dbmongo.MongoClient) *DriverRepository {
	return &DriverRepository{client: client}
}
```

- [ ] **Step 10: Verify everything still builds and passes**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: all exit 0.

- [ ] **Step 11: Commit**

```bash
git add services/driver_service/service services/driver_service/repository/mongo/driver_repository.go
git commit -m "feat(driver_service): add service layer with unit tests"
```

---

### Task 4: HTTP entities and validation

**Files:**
- Create: `services/driver_service/entity/http/requests.go`
- Create: `services/driver_service/entity/http/responses.go`
- Create: `services/driver_service/validation/validation.go`

**Interfaces:**
- Consumes: `serviceEntity.TaxiType.IsValid()`, `serviceEntity.Status.IsValid()` (Task 1).
- Produces: `http.CreateDriverReq{UserID, TaxiType}`, `http.UpdateProfileReq{TaxiType}`, `http.UpdateStatusReq{Status}`, `http.DriverResp{ID, UserID, TaxiType, Status}`, `validation.Register(v *validator.Validate) error`.

- [ ] **Step 1: Add request/response DTOs**

Create `entity/http/requests.go`:

```go
package http

type CreateDriverReq struct {
	UserID   string `json:"user_id" binding:"required" validate:"required"`
	TaxiType string `json:"taxi_type" binding:"required" validate:"required,taxi_type"`
}

type UpdateProfileReq struct {
	TaxiType string `json:"taxi_type" binding:"required" validate:"required,taxi_type"`
}

type UpdateStatusReq struct {
	Status string `json:"status" binding:"required" validate:"required,driver_status"`
}
```

Create `entity/http/responses.go`:

```go
package http

type DriverResp struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	TaxiType string `json:"taxi_type"`
	Status   string `json:"status"`
}
```

- [ ] **Step 2: Add custom validators**

Create `validation/validation.go`:

```go
package validation

import (
	"github.com/go-playground/validator/v10"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

func Register(v *validator.Validate) error {
	if err := v.RegisterValidation("taxi_type", validateTaxiType); err != nil {
		return err
	}
	return v.RegisterValidation("driver_status", validateStatus)
}

func validateTaxiType(fl validator.FieldLevel) bool {
	return serviceEntity.TaxiType(fl.Field().String()).IsValid()
}

func validateStatus(fl validator.FieldLevel) bool {
	return serviceEntity.Status(fl.Field().String()).IsValid()
}
```

- [ ] **Step 3: Add the validator dependency and verify it builds**

```bash
cd services/driver_service
go get github.com/go-playground/validator/v10@v10.30.1
go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add services/driver_service/entity/http services/driver_service/validation \
  services/driver_service/go.mod services/driver_service/go.sum
git commit -m "feat(driver_service): add HTTP DTOs and enum validators"
```

---

### Task 5: Handler layer and JWT middleware

**Files:**
- Create: `services/driver_service/handler/http/http.go`
- Create: `services/driver_service/handler/http/driver_handler.go`
- Create: `services/driver_service/handler/http/middleware/auth.go`
- Test: `services/driver_service/handler/http/middleware/auth_test.go`

**Interfaces:**
- Consumes: `service.DriverService` methods (Task 3), `http.CreateDriverReq`/`UpdateProfileReq`/`UpdateStatusReq`/`DriverResp` (Task 4), `errorsx.Err*` (Task 1), `shared/errorsx.HttpErrResp`/`ErrInternal`.
- Produces: `http.Handler` struct, `http.NewHttpHandler(s *service.DriverService, v *validator.Validate) *Handler`, methods `Profile`, `UpdateProfile`, `UpdateStatus`, `CreateDriver` (all `func(c *gin.Context)`); `middleware.Auth(secret string) gin.HandlerFunc` setting context keys `"userID"` (string) and `"roles"` (`[]string`).

- [ ] **Step 1: Add the JWT auth middleware (copied from `user_service`)**

Create `handler/http/middleware/auth.go` — identical to
`services/user_service/handler/http/middleware/auth.go`, only the import
path for `entity/service` and `shared/errorsx` changes:

```go
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Authorization header is empty"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Authorization header format is invalid"})
			return
		}

		tokenString := parts[1]
		claims := &CustomClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorsx.HttpErrResp{Message: "Unauthorized"})
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}
```

- [ ] **Step 2: Write the middleware test (copied from `user_service`, minus `RequireRole` — not needed for step 1)**

Create `handler/http/middleware/auth_test.go`:

```go
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type protectedResp struct {
	UserID string   `json:"userID"`
	Roles  []string `json:"roles"`
}

func newAuthTestRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/protected", func(c *gin.Context) {
		roles, _ := c.Get("roles")
		rolesSlice, _ := roles.([]string)
		c.JSON(http.StatusOK, protectedResp{
			UserID: c.GetString("userID"),
			Roles:  rolesSlice,
		})
	})
	return r
}

func signToken(t *testing.T, secret string, claims *CustomClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func doRequest(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth_ValidToken(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)

	token := signToken(t, secret, &CustomClaims{
		Roles: []string{"driver"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	w := doRequest(r, "Bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var got protectedResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("got userID %q, want %q", got.UserID, "user-1")
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	r := newAuthTestRouter("test-secret")
	w := doRequest(r, "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	r := newAuthTestRouter("test-secret")
	w := doRequest(r, "Token abc123")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)
	token := signToken(t, "wrong-secret", &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	w := doRequest(r, "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	r := newAuthTestRouter(secret)
	token := signToken(t, secret, &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	w := doRequest(r, "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
```

- [ ] **Step 3: Add the JWT dependency and run the middleware tests**

```bash
cd services/driver_service
go get github.com/golang-jwt/jwt/v5@v5.3.1
go get github.com/gin-gonic/gin@v1.12.0
go test ./handler/http/middleware/... -v
```

Expected: all `TestAuth_*` tests PASS.

- [ ] **Step 4: Add the `Handler` struct**

Create `handler/http/http.go`:

```go
package http

import (
	"github.com/go-playground/validator/v10"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
)

type Handler struct {
	driverSvc *service.DriverService
	validate  *validator.Validate
}

func NewHttpHandler(s *service.DriverService, v *validator.Validate) *Handler {
	return &Handler{driverSvc: s, validate: v}
}
```

- [ ] **Step 5: Implement the driver handlers**

Create `handler/http/driver_handler.go`:

```go
package http

import (
	"errors"
	"log/slog"
	"net/http"

	resp "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/http"
	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	sharedErrorsx "github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
	"github.com/gin-gonic/gin"
)

func toDriverResp(d *serviceEntity.Driver) resp.DriverResp {
	return resp.DriverResp{
		ID:       d.ID,
		UserID:   d.UserID,
		TaxiType: string(d.TaxiType),
		Status:   string(d.Status),
	}
}

func (h *Handler) Profile(c *gin.Context) {
	userID := c.GetString("userID")
	driver, err := h.driverSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverNotFound):
			c.JSON(http.StatusNotFound, sharedErrorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("get profile", "error", err)
			c.JSON(http.StatusInternalServerError, sharedErrorsx.HttpErrResp{Message: sharedErrorsx.ErrInternal.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, toDriverResp(driver))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	var req resp.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}

	driver, err := h.driverSvc.UpdateProfile(c.Request.Context(), userID, serviceEntity.TaxiType(req.TaxiType))
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverNotFound):
			c.JSON(http.StatusNotFound, sharedErrorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidTaxiType):
			c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("update profile", "error", err)
			c.JSON(http.StatusInternalServerError, sharedErrorsx.HttpErrResp{Message: sharedErrorsx.ErrInternal.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, toDriverResp(driver))
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	userID := c.GetString("userID")
	var req resp.UpdateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}

	driver, err := h.driverSvc.UpdateStatus(c.Request.Context(), userID, serviceEntity.Status(req.Status))
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverNotFound):
			c.JSON(http.StatusNotFound, sharedErrorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidStatus):
			c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("update status", "error", err)
			c.JSON(http.StatusInternalServerError, sharedErrorsx.HttpErrResp{Message: sharedErrorsx.ErrInternal.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, toDriverResp(driver))
}

func (h *Handler) CreateDriver(c *gin.Context) {
	var req resp.CreateDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		return
	}

	driver, err := h.driverSvc.CreateDriver(c.Request.Context(), serviceEntity.CreateDriverInput{
		UserID:   req.UserID,
		TaxiType: serviceEntity.TaxiType(req.TaxiType),
	})
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrDriverAlreadyExists):
			c.JSON(http.StatusConflict, sharedErrorsx.HttpErrResp{Message: err.Error()})
		case errors.Is(err, errorsx.ErrInvalidTaxiType):
			c.JSON(http.StatusUnprocessableEntity, sharedErrorsx.HttpErrResp{Message: err.Error()})
		default:
			slog.Error("create driver", "error", err)
			c.JSON(http.StatusInternalServerError, sharedErrorsx.HttpErrResp{Message: sharedErrorsx.ErrInternal.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, toDriverResp(driver))
}
```

- [ ] **Step 6: Verify it builds**

```bash
go build ./...
go vet ./...
```

Expected: exits 0.

- [ ] **Step 7: Commit**

```bash
git add services/driver_service/handler
git commit -m "feat(driver_service): add HTTP handlers and JWT middleware"
```

---

### Task 6: Wiring — `app/run.go`, `cmd/main.go`

**Files:**
- Create: `services/driver_service/app/run.go`
- Create: `services/driver_service/cmd/main.go`

**Interfaces:**
- Consumes: everything produced by Tasks 1-5.
- Produces: a runnable binary at `./cmd`.

- [ ] **Step 1: Write `app/run.go`**

Create `app/run.go` — mirrors `services/user_service/app/run.go` exactly,
adjusted for Driver Service's package names and three routes:

```go
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	dbmongo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app/db/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
	httphandler "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/handler/http"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/handler/http/middleware"
	mongomigration "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/migrations/mongo"
	mongorepo "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/repository/mongo"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/validation"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoConn, err := dbmongo.New(ctx, cfg.Mongo)
	if err != nil {
		return fmt.Errorf("connect mongo: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoConn.Close(closeCtx); err != nil {
			slog.Error("close mongo connection", "error", err)
		}
	}()

	if err := mongomigration.RunMigrations(ctx, *mongoConn); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	driverRepo := mongorepo.NewDriverRepository(mongoConn)
	driverSrv := service.NewDriverService(driverRepo)

	validate := validator.New()
	if err := validation.Register(validate); err != nil {
		return fmt.Errorf("register validators: %w", err)
	}

	h := httphandler.NewHttpHandler(driverSrv, validate)
	r := gin.Default()
	registerRoutes(r, h, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host.Host, cfg.Host.Port),
		Handler: r,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	slog.Info("server stopped gracefully")
	return nil
}

func registerRoutes(r *gin.Engine, h *httphandler.Handler, cfg *config.Config) {
	r.POST("/internal/drivers", h.CreateDriver)

	profile := r.Group("/profile")
	profile.Use(middleware.Auth(cfg.JWT.Secret))
	{
		profile.GET("", h.Profile)
		profile.PATCH("", h.UpdateProfile)
		profile.PATCH("/status", h.UpdateStatus)
	}
}
```

- [ ] **Step 2: Write `cmd/main.go`**

Create `cmd/main.go`:

```go
package main

import (
	"log"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/app"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/config"
)

func main() {
	cfg := config.Load()
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Verify the whole module builds and tests pass**

```bash
cd services/driver_service
gofmt -l .
go build ./...
go vet ./...
go test ./...
```

Expected: `gofmt -l .` prints nothing, everything else exits 0.

- [ ] **Step 4: Manual smoke test against a local Mongo**

Start a Mongo instance (reuse the one from `docker-compose.yaml` at the repo
root, or run a throwaway one), then:

```bash
cd services/driver_service
go build -o /tmp/driver_service_bin ./cmd
HTTP_PORT=18081 MONGO_DB=driver_taxi /tmp/driver_service_bin
```

In another terminal:

```bash
curl -s -w "\n%{http_code}\n" http://localhost:18081/internal/drivers \
  -X POST -H "Content-Type: application/json" \
  -d '{"user_id":"smoke-user-1","taxi_type":"economy"}'
```

Expected: `201` with a JSON body containing `"status":"offline"`.

Then send `SIGTERM` to the process (`Ctrl+C` in the terminal running it, or
`kill -TERM <pid>`) and confirm the log shows `"shutdown signal received"`
followed by `"server stopped gracefully"` before the process exits — this is
the same check already validated for `user_service`.

- [ ] **Step 5: Commit**

```bash
git add services/driver_service/app services/driver_service/cmd
git commit -m "feat(driver_service): wire app.Run with graceful shutdown"
```

---

### Task 7: Docker Compose wiring

**Files:**
- Create: `services/driver_service/.env.example`
- Create: `services/driver_service/Dockerfile`
- Modify: `docker-compose.yaml` (repo root)

**Interfaces:**
- Consumes: the binary produced by Task 6.
- Produces: a `driver_service` entry in `docker-compose.yaml`, reachable on host port `8081`.

- [ ] **Step 1: Add `.env.example`**

Create `services/driver_service/.env.example`:

```
HTTP_HOST=localhost
HTTP_PORT=8081

MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_DB=driver_taxi
MONGO_USERNAME=taxi
MONGO_PASSWORD=taxi

JWT_SECRET=taxi
```

- [ ] **Step 2: Add the Dockerfile**

Create `services/driver_service/Dockerfile` (same shape as
`services/user_service/Dockerfile` — build context must be the repo root
because of the `shared` module's relative `replace` directive):

```dockerfile
# Build context must be the repo root (services/driver_service depends on
# ../../shared via a local `replace` directive in go.mod, so the shared
# module source must be present in the build context too).
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

COPY shared/ shared/
COPY services/driver_service/ services/driver_service/

WORKDIR /workspace/services/driver_service
RUN CGO_ENABLED=0 go build -o /out/driver_service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/driver_service .

EXPOSE 8081
ENTRYPOINT ["./driver_service"]
```

- [ ] **Step 3: Add the service to `docker-compose.yaml`**

Modify `docker-compose.yaml` at the repo root — add this block as a new
entry under `services:`, alongside the existing `user_service` entry:

```yaml
  driver_service:
    build:
      context: .
      dockerfile: services/driver_service/Dockerfile
    container_name: taxi-driver-service
    restart: unless-stopped
    environment:
      HTTP_HOST: 0.0.0.0
      HTTP_PORT: 8081
      MONGO_HOST: mongo
      MONGO_PORT: 27017
      MONGO_DB: driver_taxi
      MONGO_USERNAME: ${MONGO_ROOT_USER:-taxi}
      MONGO_PASSWORD: ${MONGO_ROOT_PASSWORD:-taxi}
      JWT_SECRET: ${JWT_SECRET:-taxi}
    ports:
      - "8081:8081"
    depends_on:
      mongo:
        condition: service_healthy
```

- [ ] **Step 4: Build and smoke-test through Compose**

```bash
cd /home/user/projects/InnoTaxi
docker compose up -d --build driver_service
```

Wait for it to report healthy/running, then repeat the smoke test from
Task 6 Step 4 against port `8081` (or whichever host port you mapped):

```bash
curl -s -w "\n%{http_code}\n" http://localhost:8081/internal/drivers \
  -X POST -H "Content-Type: application/json" \
  -d '{"user_id":"smoke-user-2","taxi_type":"comfort"}'
```

Expected: `201`. Then mint a JWT signed with the same `JWT_SECRET` (`sub` =
the `user_id` you just created) and confirm:

```bash
curl -s -w "\n%{http_code}\n" http://localhost:8081/profile \
  -H "Authorization: Bearer <token>"

curl -s -w "\n%{http_code}\n" http://localhost:8081/profile/status \
  -X PATCH -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"status":"available"}'
```

Expected: both `200`, and the second response shows
`"status":"available"`.

Tear down when done:

```bash
docker compose down
```

- [ ] **Step 5: Commit**

```bash
git add services/driver_service/.env.example services/driver_service/Dockerfile docker-compose.yaml
git commit -m "feat(driver_service): add Dockerfile and docker-compose wiring"
```
