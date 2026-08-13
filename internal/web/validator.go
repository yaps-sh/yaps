package web

import (
	"fmt"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

func newValidator(maxSize int64) *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.RegisterValidation(
		"utf8", func(fl validator.FieldLevel) bool {
			return utf8.ValidString(fl.Field().String())
		},
	); err != nil {
		panic(fmt.Sprintf("failed to register utf8 validator: %v", err))
	}
	if err := v.RegisterValidation(
		"max_size", func(fl validator.FieldLevel) bool {
			return int64(len(fl.Field().String())) <= maxSize
		},
	); err != nil {
		panic(fmt.Sprintf("failed to register max_size validator: %v", err))
	}

	return v
}
