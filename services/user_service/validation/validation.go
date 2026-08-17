package validation

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/consts"
)

var phoneRegexp = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

func Register(v *validator.Validate) error {
	if err := v.RegisterValidation("phone", validatePhone); err != nil {
		return err
	}
	return v.RegisterValidation("role", validateRole)
}

func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegexp.MatchString(fl.Field().String())
}

func validateRole(fl validator.FieldLevel) bool {
	switch strings.ToLower(fl.Field().String()) {
	case consts.UserRole, consts.DriverRole:
		return true
	default:
		return false
	}
}
