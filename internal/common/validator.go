package common

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

// Errors returns the map of errors.
func (v *Validator) Error() map[string]string {
	return v.Errors
}
