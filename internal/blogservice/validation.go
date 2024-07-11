package blogservice

import (
	"regexp"

	"github.com/sushihentaime/blogist/internal/common"
)

var (
	TitleRX = regexp.MustCompile("^[a-zA-Z0-9 ]+$")
)

func validateTitle(v *common.Validator, title string) {
	v.Check(title != "", "title", "must be provided")
	v.Check(v.CheckStringLength(title, 3, 100), "title", "must be between 3 and 100 characters long")
	v.Check(TitleRX.MatchString(title), "title", "must only contain letters, numbers, and spaces")
}

func validateContent(v *common.Validator, content string) {
	v.Check(content != "", "content", "must be provided")
}

func validateInt(v *common.Validator, num int, name string) {
	v.Check(num > 0, name, "must be greater than zero")
}
