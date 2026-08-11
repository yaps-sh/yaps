package web

import (
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation(
		"utf8", func(fl validator.FieldLevel) bool {
			return utf8.ValidString(fl.Field().String())
		},
	)

	return v
}
