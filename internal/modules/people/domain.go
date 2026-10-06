// Package people manages the people the owner knows (PPL-01). Contacts are
// records, not accounts: nothing here grants a contact any access (PPL rules).
package people

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// Field limits. They match the contract in api/openapi.yaml.
const (
	maxDisplayName = 200
	maxNickname    = 100
	maxEmail       = 254
	maxPhone       = 40
	maxHowWeMet    = 500
	maxNotes       = 10000
)

// Contact is a person the owner knows. An empty optional field is unset.
type Contact struct {
	ID          uuid.UUID
	DisplayName string
	Nickname    string
	Email       string
	Phone       string
	Birthday    string // YYYY-MM-DD, or empty
	HowWeMet    string
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int32
}

// Input is the writable part of a contact.
type Input struct {
	DisplayName string
	Nickname    string
	Email       string
	Phone       string
	Birthday    string
	HowWeMet    string
	Notes       string
}

// Patch is a JSON Merge Patch over the writable fields. A missing key keeps
// the current value; a nil value clears it.
type Patch map[string]*string

// Page is one page of contacts. Next is set when more remain.
type Page struct {
	Items []Contact
	Next  *uuid.UUID
}

// normalized trims surrounding whitespace so "  " counts as empty.
func (in Input) normalized() Input {
	return Input{
		DisplayName: strings.TrimSpace(in.DisplayName),
		Nickname:    strings.TrimSpace(in.Nickname),
		Email:       strings.TrimSpace(in.Email),
		Phone:       strings.TrimSpace(in.Phone),
		Birthday:    strings.TrimSpace(in.Birthday),
		HowWeMet:    strings.TrimSpace(in.HowWeMet),
		Notes:       strings.TrimSpace(in.Notes),
	}
}

// validate returns an Invalid error naming every field that needs attention.
func (in Input) validate() error {
	fields := map[string]string{}
	limit := func(field, value string, max int, label string) {
		if utf8.RuneCountInString(value) > max {
			fields[field] = fmt.Sprintf("%s can be at most %d characters.", label, max)
		}
	}

	if in.DisplayName == "" {
		fields["display_name"] = "Enter a name."
	}
	limit("display_name", in.DisplayName, maxDisplayName, "Name")
	limit("nickname", in.Nickname, maxNickname, "Nickname")
	limit("email", in.Email, maxEmail, "Email")
	limit("phone", in.Phone, maxPhone, "Phone")
	limit("how_we_met", in.HowWeMet, maxHowWeMet, "How you met")
	limit("notes", in.Notes, maxNotes, "Notes")

	if in.Birthday != "" {
		if _, err := time.Parse(time.DateOnly, in.Birthday); err != nil {
			fields["birthday"] = "Use the format YYYY-MM-DD."
		}
	}

	if len(fields) > 0 {
		return invalid(fields)
	}
	return nil
}

// apply returns cur with the patch's keys changed. Unknown keys are rejected
// so a typo can't silently do nothing.
func (p Patch) apply(cur Input) (Input, error) {
	unknown := map[string]string{}
	for key, value := range p {
		v := ""
		if value != nil {
			v = *value
		}
		switch key {
		case "display_name":
			cur.DisplayName = v
		case "nickname":
			cur.Nickname = v
		case "email":
			cur.Email = v
		case "phone":
			cur.Phone = v
		case "birthday":
			cur.Birthday = v
		case "how_we_met":
			cur.HowWeMet = v
		case "notes":
			cur.Notes = v
		default:
			unknown[key] = "This field can't be changed."
		}
	}
	if len(unknown) > 0 {
		return Input{}, invalid(unknown)
	}
	return cur, nil
}

func invalid(fields map[string]string) error {
	return apperr.Invalid("people.invalid_contact", "Some fields need attention.", fields)
}

// nullable turns "" into nil, for columns where NULL means unset.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// stringOf reads a nullable column as "" when NULL.
func stringOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
