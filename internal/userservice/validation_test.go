package userservice

import (
	"testing"

	"github.com/sushihentaime/blogist/internal/common"
)

func TestValidateUsername(t *testing.T) {
	testCases := []struct {
		username string
		valid    bool
	}{
		{username: "", valid: false},
		{username: "a", valid: false},
		{username: "ab", valid: false},
		{username: "abc", valid: true},
		{username: "abcd", valid: true},
		{username: "valid123", valid: true},
		{username: "invalid!", valid: false},
		{username: "invalid username", valid: false},
		{username: "invalid-username", valid: false},
		{username: "invalid_username", valid: false},
		{username: "invalid.username", valid: false},
		{username: "abcdefghijklmnopqrstuvwxyz", valid: false},
	}

	for _, tc := range testCases {
		t.Run(tc.username, func(t *testing.T) {
			v := common.NewValidator()
			validateUsername(v, tc.username)
			if v.Valid() != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, v.Valid())
				// print the errors
				for _, e := range v.Errors {
					t.Log(e)
				}
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	testCases := []struct {
		email string
		valid bool
	}{
		{email: "", valid: false},
		{email: "a", valid: false},
		{email: "a@", valid: false},
		{email: "a@b", valid: false},
		{email: "a@b.c", valid: false},
		{email: "a@b.com", valid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			v := common.NewValidator()
			validateEmail(v, tc.email)
			if v.Valid() != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, v.Valid())
				// print the errors
				for _, e := range v.Errors {
					t.Log(e)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	testCases := []struct {
		password string
		valid    bool
	}{
		{password: "", valid: false},
		{password: "a", valid: false},
		{password: "ab", valid: false},
		{password: "abc", valid: false},
		{password: "abcd", valid: false},
		{password: "abcde", valid: false},
		{password: "abcdef", valid: false},
		{password: "password123", valid: false},
		{password: "Password123", valid: false},
		{password: "Password!23", valid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.password, func(t *testing.T) {
			v := common.NewValidator()
			validatePassword(v, tc.password)
			if v.Valid() != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, v.Valid())
				// print the errors
				for _, e := range v.Errors {
					t.Log(e)
				}
			}
		})
	}
}
