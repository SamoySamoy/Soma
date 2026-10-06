// Package bodymap assembles the landing page: every area of life with its
// status. Modules report their own status through a Provider; this package
// only arranges the results (tech spec section 12).
package bodymap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

// Area is one region of the body map. The values match the contract's enum.
type Area string

// Areas, in the order the map lists them.
const (
	AreaMind           Area = "mind"
	AreaSelf           Area = "self"
	AreaResponsibility Area = "responsibilities"
	AreaHeart          Area = "heart"
	AreaBody           Area = "body"
	AreaWork           Area = "work"
	AreaMoney          Area = "money"
	AreaGrowth         Area = "growth"
	AreaJourneys       Area = "journeys"
	AreaHome           Area = "home"
	AreaPapers         Area = "papers"
)

// Level is how an area needs attention. Unknown means the area isn't enabled.
type Level string

// Levels, from quietest to most urgent. Unknown is for areas not built yet.
const (
	LevelCalm      Level = "calm"
	LevelAttention Level = "attention"
	LevelUrgent    Level = "urgent"
	LevelUnknown   Level = "unknown"
)

// Phase is the release that delivers an area (business spec section 19).
type Phase string

// Phases.
const (
	PhaseP1 Phase = "P1"
	PhaseP2 Phase = "P2"
	PhaseP3 Phase = "P3"
)

// Definition describes one area. The registry lists them in map order.
type Definition struct {
	Area  Area
	Phase Phase
}

// Registry is the complete list of areas. Adding a module adds its area here
// and registers a provider; nothing else changes.
var Registry = []Definition{
	{AreaMind, PhaseP1},
	{AreaSelf, PhaseP1},
	{AreaResponsibility, PhaseP1},
	{AreaHeart, PhaseP1},
	{AreaBody, PhaseP2},
	{AreaWork, PhaseP2},
	{AreaMoney, PhaseP1},
	{AreaGrowth, PhaseP2},
	{AreaJourneys, PhaseP3},
	{AreaHome, PhaseP2},
	{AreaPapers, PhaseP2},
}

// Result is what a module reports for its area.
type Result struct {
	Level  Level
	Counts map[string]int
}

// Provider reports the status of one area. It runs inside the snapshot's
// transaction, so every area is read from the same state.
type Provider interface {
	Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, now time.Time) (Result, error)
}

// AreaStatus is one area as shown on the map.
type AreaStatus struct {
	Area    Area
	Phase   Phase
	Level   Level
	Enabled bool
	Counts  map[string]int
}

// Snapshot is the whole map at one moment.
type Snapshot struct {
	AsOf  time.Time
	Areas []AreaStatus
}

// Service builds snapshots. Each area's provider is looked up by area.
type Service struct {
	db        *db.DB
	clock     clock.Clock
	providers map[Area]Provider
}

// NewService returns a Service. providers maps each enabled area to its provider.
func NewService(d *db.DB, clk clock.Clock, providers map[Area]Provider) *Service {
	return &Service{db: d, clock: clk, providers: providers}
}

// Snapshot returns the map for the space on ctx.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	spaceID, ok := space.FromContext(ctx)
	if !ok {
		return Snapshot{}, fmt.Errorf("bodymap snapshot: no space on context")
	}
	now := s.clock.Now()
	out := Snapshot{AsOf: now, Areas: make([]AreaStatus, 0, len(Registry))}
	err := s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.BodymapRead); err != nil {
			return err
		}
		for _, def := range Registry {
			st := AreaStatus{Area: def.Area, Phase: def.Phase, Level: LevelUnknown, Counts: map[string]int{}}
			if p, ok := s.providers[def.Area]; ok {
				res, err := p.Status(ctx, tx, spaceID, now)
				if err != nil {
					return fmt.Errorf("status of %s: %w", def.Area, err)
				}
				st.Enabled = true
				st.Level = res.Level
				st.Counts = res.Counts
			}
			out.Areas = append(out.Areas, st)
		}
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	return out, nil
}
