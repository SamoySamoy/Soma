// Package self holds the owner's Self profile (SELF-01, SELF-03, SELF-04): one
// per space, created with the space. It is what the Face of the body map shows.
package self

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	"github.com/SamoySamoy/Soma/internal/bodymap"
	sstore "github.com/SamoySamoy/Soma/internal/modules/self/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

// Field limits. They match the contract in api/openapi.yaml.
const (
	maxPreferredName = 100
	maxCoreValues    = 2000
	maxBio           = 5000
)

// Profile is the Self profile. An empty field is unset.
type Profile struct {
	PreferredName string
	BirthDate     string // YYYY-MM-DD, or empty
	CoreValues    string
	Bio           string
	UpdatedAt     time.Time
	Version       int32
}

// Input is the writable part of a profile.
type Input struct {
	PreferredName string
	BirthDate     string
	CoreValues    string
	Bio           string
}

// Patch is a JSON Merge Patch over the writable fields. Values are string or nil.
type Patch map[string]any

// Service reads and edits the Self profile of the space on the context.
type Service struct {
	db    *db.DB
	clock clock.Clock
}

// NewService returns a Service backed by d.
func NewService(d *db.DB, clk clock.Clock) *Service {
	return &Service{db: d, clock: clk}
}

// Get returns the profile.
func (s *Service) Get(ctx context.Context) (Profile, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Profile{}, err
	}
	var out Profile
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.SelfRead); err != nil {
			return err
		}
		out, err = load(ctx, tx, spaceID)
		return err
	})
	return out, err
}

// Update applies patch to the profile. expectedVersion must equal the stored
// version; otherwise the update is refused with 412.
func (s *Service) Update(ctx context.Context, expectedVersion int32, patch Patch) (Profile, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Profile{}, err
	}
	var out Profile
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.SelfWrite); err != nil {
			return err
		}
		cur, err := load(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return staleVersion()
		}
		next, err := patch.apply(cur.input())
		if err != nil {
			return err
		}
		next = next.normalized()
		if err := next.validate(); err != nil {
			return err
		}
		bday, err := birthOf(next.BirthDate)
		if err != nil {
			return err
		}
		_, err = sstore.New(tx).UpdateSelfProfile(ctx, sstore.UpdateSelfProfileParams{
			SpaceID:         spaceID,
			PreferredName:   nullable(next.PreferredName),
			BirthDate:       bday,
			CoreValues:      nullable(next.CoreValues),
			Bio:             nullable(next.Bio),
			Now:             s.clock.Now(),
			ExpectedVersion: expectedVersion,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return staleVersion()
		}
		if err != nil {
			return fmt.Errorf("update self profile: %w", err)
		}
		out, err = load(ctx, tx, spaceID)
		return err
	})
	return out, err
}

func spaceFrom(ctx context.Context) (uuid.UUID, error) {
	id, ok := space.FromContext(ctx)
	if !ok {
		return uuid.Nil, apperr.Unauthorized("space.required", "Choose a space to continue.")
	}
	return id, nil
}

func load(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID) (Profile, error) {
	r, err := sstore.New(tx).GetSelfProfile(ctx, spaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, apperr.NotFound("self.profile_not_found", "Profile not found.")
	}
	if err != nil {
		return Profile{}, fmt.Errorf("load self profile: %w", err)
	}
	return Profile{
		PreferredName: stringOf(r.PreferredName),
		BirthDate:     dateOf(r.BirthDate),
		CoreValues:    stringOf(r.CoreValues),
		Bio:           stringOf(r.Bio),
		UpdatedAt:     r.UpdatedAt,
		Version:       r.Version,
	}, nil
}

func (p Profile) input() Input {
	return Input{PreferredName: p.PreferredName, BirthDate: p.BirthDate, CoreValues: p.CoreValues, Bio: p.Bio}
}

func (in Input) normalized() Input {
	return Input{
		PreferredName: strings.TrimSpace(in.PreferredName),
		BirthDate:     strings.TrimSpace(in.BirthDate),
		CoreValues:    strings.TrimSpace(in.CoreValues),
		Bio:           strings.TrimSpace(in.Bio),
	}
}

func (in Input) validate() error {
	fields := map[string]string{}
	if utf8.RuneCountInString(in.PreferredName) > maxPreferredName {
		fields["preferred_name"] = fmt.Sprintf("Name can be at most %d characters.", maxPreferredName)
	}
	if utf8.RuneCountInString(in.CoreValues) > maxCoreValues {
		fields["core_values"] = fmt.Sprintf("Values can be at most %d characters.", maxCoreValues)
	}
	if utf8.RuneCountInString(in.Bio) > maxBio {
		fields["bio"] = fmt.Sprintf("Bio can be at most %d characters.", maxBio)
	}
	if in.BirthDate != "" {
		if _, err := time.Parse(time.DateOnly, in.BirthDate); err != nil {
			fields["birth_date"] = "Use the format YYYY-MM-DD."
		}
	}
	if len(fields) > 0 {
		return apperr.Invalid("self.invalid_profile", "Some fields need attention.", fields)
	}
	return nil
}

// apply returns cur with the patch applied. Unknown keys and wrong types are refused.
func (p Patch) apply(cur Input) (Input, error) {
	bad := map[string]string{}
	for key, v := range p {
		s, isText := textOf(v)
		if key != "preferred_name" && key != "birth_date" && key != "core_values" && key != "bio" {
			bad[key] = "This field can't be changed."
			continue
		}
		if !isText {
			bad[key] = "Use text."
			continue
		}
		switch key {
		case "preferred_name":
			cur.PreferredName = s
		case "birth_date":
			cur.BirthDate = s
		case "core_values":
			cur.CoreValues = s
		case "bio":
			cur.Bio = s
		}
	}
	if len(bad) > 0 {
		return Input{}, apperr.Invalid("self.invalid_profile", "Some fields need attention.", bad)
	}
	return cur, nil
}

func textOf(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", true
	case string:
		return x, true
	default:
		return "", false
	}
}

// Fields counts the profile fields that have something in them.
func (p Profile) Fields() (filled, total int) {
	for _, s := range []string{p.PreferredName, p.BirthDate, p.CoreValues, p.Bio} {
		if s != "" {
			filled++
		}
	}
	return filled, 4
}

// BodymapProvider reports the Self area: attention while any profile field is
// still empty, calm once the profile is filled in.
type BodymapProvider struct{}

// Status implements bodymap.Provider.
func (BodymapProvider) Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, _ time.Time) (bodymap.Result, error) {
	p, err := load(ctx, tx, spaceID)
	if err != nil {
		return bodymap.Result{}, err
	}
	filled, total := p.Fields()
	level := bodymap.LevelCalm
	if filled < total {
		level = bodymap.LevelAttention
	}
	return bodymap.Result{Level: level, Counts: map[string]int{"filled": filled, "fields": total}}, nil
}

func staleVersion() error {
	return apperr.New(apperr.KindPreconditionFailed, "self.stale_version",
		"Your profile was changed elsewhere. Reload it and try again.")
}

func birthOf(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, apperr.Invalid("self.invalid_profile", "Some fields need attention.",
			map[string]string{"birth_date": "Use the format YYYY-MM-DD."})
	}
	return &t, nil
}

func dateOf(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
