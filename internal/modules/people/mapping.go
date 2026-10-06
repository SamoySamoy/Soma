package people

import (
	"time"

	pstore "github.com/SamoySamoy/Soma/internal/modules/people/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// input returns the writable fields of c, for applying a patch.
func (c Contact) input() Input {
	return Input{
		DisplayName: c.DisplayName,
		Nickname:    c.Nickname,
		Email:       c.Email,
		Phone:       c.Phone,
		Birthday:    c.Birthday,
		HowWeMet:    c.HowWeMet,
		Notes:       c.Notes,
	}
}

func fromGetRow(r pstore.GetContactRow) Contact {
	return Contact{
		ID:          r.ID,
		DisplayName: r.DisplayName,
		Nickname:    stringOf(r.Nickname),
		Email:       stringOf(r.Email),
		Phone:       stringOf(r.Phone),
		Birthday:    dateOf(r.Birthday),
		HowWeMet:    stringOf(r.HowWeMet),
		Notes:       stringOf(r.Notes),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Version:     r.Version,
	}
}

func fromListRow(r pstore.ListContactsRow) Contact {
	return Contact{
		ID:          r.ID,
		DisplayName: r.DisplayName,
		Nickname:    stringOf(r.Nickname),
		Email:       stringOf(r.Email),
		Phone:       stringOf(r.Phone),
		Birthday:    dateOf(r.Birthday),
		HowWeMet:    stringOf(r.HowWeMet),
		Notes:       stringOf(r.Notes),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Version:     r.Version,
	}
}

// birthdayOf parses a validated YYYY-MM-DD string for storage; "" stays NULL.
func birthdayOf(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, apperr.Invalid("people.invalid_contact", "Some fields need attention.",
			map[string]string{"birthday": "Use the format YYYY-MM-DD."})
	}
	return &t, nil
}

func dateOf(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

func ptrTo[T any](v T) *T { return &v }
