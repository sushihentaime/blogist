package userservice

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func (p *Password) set(pwd string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		return err
	}

	p.Plain = pwd
	p.hash = hash

	return nil
}

func (p *Password) compare(pwd string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(pwd))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}
