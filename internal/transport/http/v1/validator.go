package v1

import "github.com/go-playground/validator/v10"

type structValidator struct {
	validate *validator.Validate
}

type Validator interface {
	Validate(out any) error
}

func NewValidator() Validator {
	return &structValidator{
		validate: validator.New(),
	}
}

func (v *structValidator) Validate(out any) error {
	return v.validate.Struct(out) //nolint: wrapcheck
}
