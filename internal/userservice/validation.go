package userservice

import (
	"regexp"

	"github.com/sushihentaime/blogist/internal/common"
)

var (
	EmailRX     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	UsernameRX  = regexp.MustCompile("^[a-zA-Z0-9]+$")
	UppercaseRX = regexp.MustCompile("[A-Z]")
	LowercaseRX = regexp.MustCompile("[a-z]")
	NumberRX    = regexp.MustCompile("[0-9]")
	SymbolRX    = regexp.MustCompile(`[#?!@$%^&*_\\-]`)
)

func validateUsername(v *common.Validator, username string) {
	v.Check(username != "", "username", "must be provided")
	v.Check(v.CheckStringLength(username, 3, 25), "username", "must be between 3 and 25 characters long")
	v.Check(UsernameRX.MatchString(username), "username", "must only contain letters and numbers")
}

func validateEmail(v *common.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(EmailRX.MatchString(email), "email", "must be a valid email address")
}

func validatePassword(v *common.Validator, password string) {
	v.Check(password != "", "password", "must be provided")

	value := v.CheckStringLength(password, 8, 72) && UppercaseRX.MatchString(password) && LowercaseRX.MatchString(password) && NumberRX.MatchString(password) && SymbolRX.MatchString(password)
	v.Check(value, "password", "must be between 8 and 72 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one symbol")
}

func ValidateToken(v *common.Validator, token string) {
	v.Check(token != "", "token", "must be provided")
	v.Check(len(token) == 26, "token", "invalid token")
}

func validateInt(v *common.Validator, num int, name string) {
	v.Check(num > 0, name, "must be greater than zero")
}
