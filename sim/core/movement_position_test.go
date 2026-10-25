package core

import (
	"math"
	"testing"
	"time"
)

// An event during a walk reads where the unit is at that moment, with the rotation parked so nothing
// else asks for the position.
func TestWalkPositionIsCurrentAtEveryEvent(t *testing.T) {
	sim, fc := setupSwingResetSim(t) // at 0 yd, one second in
	fc.MoveTo(12, sim)
	fc.WaitUntil(sim, NeverExpires)

	seen := -1.0
	sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
		DoAt:     time.Millisecond * 1500,
		OnAction: func(*Simulation) { seen = fc.DistanceFromTarget },
	}))
	stepSwingSim(t, sim, time.Millisecond*1500)

	if expected := 3.5; math.Abs(seen-expected) > 1e-9 {
		t.Fatalf("expected the event to read %.2f yd, got %.4f", expected, seen)
	}
}

// An instant move puts the unit at the distance with no walk, ends a walk under way there, and
// starts or stops the melee swings for the new distance.
func TestMoveInstantly(t *testing.T) {
	t.Run("out of melee range at once, with no walk", func(t *testing.T) {
		sim, fc := setupSwingResetSim(t) // at 0 yd, MH due at 2.6s
		aa := &fc.AutoAttacks
		mhPrevious := aa.MainhandPreviousSwing()
		var kinds []MovementUpdateType
		var positions []float64
		fc.RegisterMovementCallback(func(_ *Simulation, position float64, kind MovementUpdateType) {
			kinds = append(kinds, kind)
			positions = append(positions, position)
		})

		fc.MoveInstantlyTo(12, sim)
		if fc.DistanceFromTarget != 12 || fc.Moving {
			t.Fatalf("expected the unit standing at 12 yd, got %.2f yd and moving %v", fc.DistanceFromTarget, fc.Moving)
		}
		if len(kinds) != 1 || kinds[0] != MovementEnd || positions[0] != 12 {
			t.Fatalf("expected one MovementEnd at 12 yd, got kinds %v at %v", kinds, positions)
		}

		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, time.Second*4)
		if aa.MainhandPreviousSwing() != mhPrevious {
			t.Fatalf("expected no swing from out of range, got MH at %v", aa.MainhandPreviousSwing())
		}
	})

	t.Run("a walk under way ends where the unit lands", func(t *testing.T) {
		sim, fc := setupSwingResetSim(t)
		fc.MoveTo(12, sim) // arrives after 1.714s
		stepSwingSim(t, sim, sim.CurrentTime+time.Second)
		var kinds []MovementUpdateType
		fc.RegisterMovementCallback(func(_ *Simulation, _ float64, kind MovementUpdateType) {
			kinds = append(kinds, kind)
		})

		fc.MoveInstantlyTo(0, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+time.Second*2)

		if fc.DistanceFromTarget != 0 || fc.Moving {
			t.Fatalf("expected the unit standing at 0 yd, got %.2f yd and moving %v", fc.DistanceFromTarget, fc.Moving)
		}
		if len(kinds) != 1 || kinds[0] != MovementEnd {
			t.Fatalf("expected one MovementEnd, got %v", kinds)
		}
		if !fc.AutoAttacks.mh.enabled {
			t.Fatalf("expected the melee swing back on in range")
		}
	})
}

// A walk across melee range stops or starts the melee swings on the line. The rotation is parked, as
// under a GCD, so nothing else asks for the position on the way.
func TestWalkAcrossMeleeRange(t *testing.T) {
	speed := 7.0 // a player's run speed, yards per second
	toLine := func(yards float64) time.Duration { return DurationFromSeconds(yards / speed) }

	t.Run("no swing lands after the walk out crosses the line", func(t *testing.T) {
		sim, fc := setupSwingResetSim(t) // at 0 yd, MH due at 2.6s, OH at 2.7s
		aa := &fc.AutoAttacks
		mhPrevious, ohPrevious := aa.MainhandPreviousSwing(), aa.OffhandPreviousSwing()

		fc.MoveTo(12, sim) // crosses 5 yd at 1.714s, arrives at 2.714s
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+toLine(12))

		if aa.MainhandPreviousSwing() != mhPrevious || aa.OffhandPreviousSwing() != ohPrevious {
			t.Fatalf("expected no swing from out of range, got MH at %v and OH at %v", aa.MainhandPreviousSwing(), aa.OffhandPreviousSwing())
		}
	})

	t.Run("a swing that is due lands where the walk in crosses the line", func(t *testing.T) {
		sim, fc := setupSwingResetSim(t)
		aa := &fc.AutoAttacks
		fc.MoveTo(12, sim)
		stepSwingSim(t, sim, time.Second*3) // out of range since 1.714s, the MH swing is due

		start := sim.CurrentTime
		fc.MoveTo(0, sim) // crosses 5 yd after 1s, arrives after 1.714s
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, start+toLine(12))

		onLine := start + toLine(7)
		if swing := aa.MainhandPreviousSwing(); swing < onLine || swing > onLine+time.Millisecond {
			t.Fatalf("expected the MH swing on the line at %v, got %v", onLine, swing)
		}
	})
}
