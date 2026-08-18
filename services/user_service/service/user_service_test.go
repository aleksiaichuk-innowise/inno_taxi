package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	serviceEntity "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/errorsx"
)

type fakeUserRepository struct {
	user *serviceEntity.User
	err  error

	updateUser   *serviceEntity.User
	updateErr    error
	updateCalled bool

	setPasswordErr    error
	setPasswordCalled bool
	setPasswordArg    string

	deleteErr    error
	deleteCalled bool

	addRoleErr    error
	addRoleCalled bool
	addRoleArg    string
}

func (f *fakeUserRepository) CreateUser(_ context.Context, _ *serviceEntity.User) (*serviceEntity.User, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeUserRepository) FindByLogin(_ context.Context, _ string) (*serviceEntity.User, error) {
	return f.user, f.err
}
func (f *fakeUserRepository) FindByID(_ context.Context, _ string) (*serviceEntity.User, error) {
	return f.user, f.err
}

func (f *fakeUserRepository) UpdateProfile(_ context.Context, _, _, _, _ string) (*serviceEntity.User, error) {
	f.updateCalled = true
	return f.updateUser, f.updateErr
}
func (f *fakeUserRepository) SetPassword(_ context.Context, _ string, password string) error {
	f.setPasswordCalled = true
	f.setPasswordArg = password
	return f.setPasswordErr
}
func (f *fakeUserRepository) DeleteProfile(_ context.Context, _ string) error {
	f.deleteCalled = true
	return f.deleteErr
}

func (f *fakeUserRepository) AddRole(_ context.Context, _ string, role string) error {
	f.addRoleCalled = true
	f.addRoleArg = role
	return f.addRoleErr
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func TestUserService_VerifyCredentials_Success(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
			Roles:        []serviceEntity.Role{serviceEntity.RoleUser},
		},
	}
	svc := NewUserService(repo)

	got, err := svc.VerifyCredentials(context.Background(), "user@example.com", "correct-password")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got user ID %q, want %q", got.ID, "1")
	}
}

func TestUserService_VerifyCredentials_WrongPassword(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			Email:        "user@example.com",
			PasswordHash: hashPassword(t, "correct-password"),
		},
	}
	svc := NewUserService(repo)

	_, err := svc.VerifyCredentials(context.Background(), "user@example.com", "wrong-password")

	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidCredentials)
	}
}

func TestUserService_VerifyCredentials_UserNotFound(t *testing.T) {
	repo := &fakeUserRepository{err: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	_, err := svc.VerifyCredentials(context.Background(), "missing@example.com", "any-password")

	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidCredentials)
	}
}

func TestUserService_GetProfile_Success(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:    "1",
			Name:  "Jane Doe",
			Email: "user@example.com",
			Roles: []serviceEntity.Role{serviceEntity.RoleUser},
		},
	}
	svc := NewUserService(repo)

	got, err := svc.GetProfile(context.Background(), "1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got user ID %q, want %q", got.ID, "1")
	}
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	repo := &fakeUserRepository{err: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	_, err := svc.GetProfile(context.Background(), "missing-id")

	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	updated := &serviceEntity.User{ID: "1", Name: "New Name", Email: "new@example.com", Phone: "+15550001111"}
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			PasswordHash: hashPassword(t, "correct-password"),
		},
		updateUser: updated,
	}
	svc := NewUserService(repo)

	got, err := svc.UpdateProfile(context.Background(), "1", serviceEntity.ProfileInput{
		Name:  "New Name",
		Email: "new@example.com",
		Phone: "+15550001111",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != updated {
		t.Errorf("got user %+v, want %+v", got, updated)
	}
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	repo := &fakeUserRepository{updateErr: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	_, err := svc.UpdateProfile(context.Background(), "missing-id", serviceEntity.ProfileInput{
		Name:  "New Name",
		Email: "new@example.com",
		Phone: "+15550001111",
	})

	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserService_UpdatePassword_Success(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			PasswordHash: hashPassword(t, "old-password"),
		},
	}
	svc := NewUserService(repo)

	err := svc.UpdatePassword(context.Background(), "1", "old-password", "new-password")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.setPasswordCalled {
		t.Fatal("expected repository SetPassword to be called")
	}
	if bcrypt.CompareHashAndPassword([]byte(repo.setPasswordArg), []byte("new-password")) != nil {
		t.Error("stored hash does not match the new password")
	}
}

func TestUserService_UpdatePassword_WrongCurrentPassword(t *testing.T) {
	repo := &fakeUserRepository{
		user: &serviceEntity.User{
			ID:           "1",
			PasswordHash: hashPassword(t, "old-password"),
		},
	}
	svc := NewUserService(repo)

	err := svc.UpdatePassword(context.Background(), "1", "wrong-password", "new-password")

	if !errors.Is(err, errorsx.ErrInvalidCredentials) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidCredentials)
	}
	if repo.setPasswordCalled {
		t.Error("expected repository SetPassword not to be called after failed password check")
	}
}

func TestUserService_UpdatePassword_NotFound(t *testing.T) {
	repo := &fakeUserRepository{err: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	err := svc.UpdatePassword(context.Background(), "missing-id", "old-password", "new-password")

	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserService_DeleteProfile_Success(t *testing.T) {
	repo := &fakeUserRepository{}
	svc := NewUserService(repo)

	err := svc.DeleteProfile(context.Background(), "1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.deleteCalled {
		t.Error("expected repository DeleteProfile to be called")
	}
}

func TestUserService_DeleteProfile_NotFound(t *testing.T) {
	repo := &fakeUserRepository{deleteErr: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	err := svc.DeleteProfile(context.Background(), "missing-id")

	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}

func TestUserService_AssignRole_Success(t *testing.T) {
	repo := &fakeUserRepository{}
	svc := NewUserService(repo)

	err := svc.AssignRole(context.Background(), "1", serviceEntity.RoleAnalyst)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.addRoleCalled {
		t.Fatal("expected repository AddRole to be called")
	}
	if repo.addRoleArg != string(serviceEntity.RoleAnalyst) {
		t.Errorf("got role %q, want %q", repo.addRoleArg, serviceEntity.RoleAnalyst)
	}
}

func TestUserService_AssignRole_InvalidRole(t *testing.T) {
	repo := &fakeUserRepository{}
	svc := NewUserService(repo)

	err := svc.AssignRole(context.Background(), "1", serviceEntity.Role("bogus"))

	if !errors.Is(err, errorsx.ErrInvalidRole) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrInvalidRole)
	}
	if repo.addRoleCalled {
		t.Error("expected repository AddRole not to be called for an invalid role")
	}
}

func TestUserService_AssignRole_NotFound(t *testing.T) {
	repo := &fakeUserRepository{addRoleErr: errorsx.ErrUserNotFound}
	svc := NewUserService(repo)

	err := svc.AssignRole(context.Background(), "missing-id", serviceEntity.RoleAnalyst)

	if !errors.Is(err, errorsx.ErrUserNotFound) {
		t.Errorf("got error %v, want %v", err, errorsx.ErrUserNotFound)
	}
}
