package common

import "fmt"

type ValidationError struct {
	Errors map[string]string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation errors: %+v", e.Errors)
}

type Validator struct {
	Errors map[string]string
}

func NewValidator() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(field, message string) {
	if _, ok := v.Errors[field]; !ok {
		v.Errors[field] = message
	}
}

func (v *Validator) Check(ok bool, field, message string) {
	if !ok {
		v.AddError(field, message)
	}
}

func (v *Validator) CheckStringLength(s string, min, max int) bool {
	return len(s) >= min && len(s) <= max
}

func (v *Validator) ValidationError() error {
	return ValidationError{Errors: v.Errors}
}
