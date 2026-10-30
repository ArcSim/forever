package core

import (
	"math"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// Arc's walk rules, all behind ForeverWalkRules:
//   - a move reaction before each walk, during which no spell with a cast time can start (the walk would cancel it);
//   - a walk back out can be decided on the way in once inside melee range: the reaction runs during the rest of the
//     walk and the walk out starts on arrival;
//   - Auto Shot fires 0.6 s after a walk ends, the windup included.

// The move reaction of the site's builds and default.
var ForeverMoveDelay = 100 * time.Millisecond

// From the end of a walk to the Auto Shot it allows, windup included (Arc's logs: 0.57 to 0.73 s).
const ForeverAutoShotStandStill = 600 * time.Millisecond

// A walk advancing a whole yard at a time, each yard one yard's walking time after the last (off: the walk is continuous,
// so past 5 yd nothing melee lands and inside 11 yd no shot goes, from the first instant).
var ForeverYardSteps = false

// Where the walk's whole yards so far have taken the unit.
func (action *MovementAction) arcSteppedPosition(sim *Simulation) float64 {
	walked := math.Abs(float64(sim.CurrentTime-action.startTime) * action.speed / float64(time.Second))
	yards := math.Min(math.Floor(walked+1e-9), math.Abs(action.moveDistance))
	return action.srcPosition + math.Copysign(yards, action.moveDistance)
}

// The yards the Movement aura logs for the timeline: 5 or less exactly while a melee swing can land and under 11 exactly
// while Auto Shot can't fire, so its range lines fall where the engine's state changes, not a yard early from rounding
// down a walk in.
func (unit *Unit) arcLoggedYards() int32 {
	d := unit.DistanceFromTarget
	switch {
	case unit.AutoAttacks.mh.IsInRange() && !(ForeverWalkRules && unit.arcWalkingOut()):
		return int32(min(max(math.Ceil(d-1e-9), 1), MaxMeleeRange))
	case d < MinRangedRange:
		return int32(min(max(math.Ceil(d-1e-9), MaxMeleeRange+1), MinRangedRange-1))
	}
	return int32(math.Floor(d + 1e-9))
}

// A run speed change during a walk: the rest of the walk goes at the new speed, and with yard steps the yard under way
// keeps the share already walked. A move to the walk's own end would be ignored as a walk already under way.
func (unit *Unit) arcRepace(sim *Simulation) {
	action := unit.movementAction
	dest := action.srcPosition + action.moveDistance
	speed := unit.GetMovementSpeed()
	var back time.Duration
	if ForeverYardSteps {
		walked := math.Abs(float64(sim.CurrentTime-action.startTime) * action.speed / float64(time.Second))
		back = DurationFromSeconds(max(0, walked-math.Floor(walked+1e-9)) / speed)
	}
	unit.UpdatePosition(sim, false)
	remaining := dest - unit.DistanceFromTarget
	if remaining == 0 {
		return
	}
	start := sim.CurrentTime - back
	registerMovementActionFrom(unit, sim, math.Copysign(speed, remaining), start, start+DurationFromSeconds(math.Abs(remaining)/speed), remaining, false)
}

// How long into the walk the unit crosses the melee edge.
func (action *MovementAction) arcEdgeDelay(edge float64) time.Duration {
	src := action.srcPosition
	if !(ForeverWalkRules && ForeverYardSteps) {
		return DurationFromSeconds((edge - src) / action.speed)
	}
	yards := math.Ceil(src - edge - 1e-9)
	if action.speed > 0 {
		// a walk out leaves on the yard that reaches the edge (arcWalkingOut)
		yards = math.Max(1, math.Ceil(edge-src-1e-9))
	}
	return DurationFromSeconds(yards / math.Abs(action.speed))
}

type arcWalk struct {
	pending          bool // a walk decided, its move reaction still running
	queued           bool // a walk decided during a walk, to start on arrival
	queuedRange      float64
	queuedByHand     bool
	pendingRange     float64 // where the walk waiting on its move reaction goes
	swungWhileMoving bool    // a melee swing landed during the walk under way

	// For the log only: when and where the walk under way (or the last one) began.
	startedAt     time.Duration
	startDistance float64
}

// The move reaction as a log-only cast (no spell, no effects), so the timeline can draw it in front of the walk. Tag 1
// is the move spell's split for walks a spell or a pet causes.
var MoveReactionID = ActionID{OtherID: proto.OtherAction_OtherActionMove, Tag: 2}

func (unit *Unit) logMoveReaction(sim *Simulation, delay time.Duration) {
	if sim.Log != nil {
		unit.Log(sim, "Casting %s (Cost = 0.000, Cast Time = %s, GCD = 0s, Effective Time = 0s)", MoveReactionID, delay)
	}
}

func (unit *Unit) logMoveReactionDone(sim *Simulation) {
	if sim.Log != nil {
		unit.Log(sim, "Completed cast %s", MoveReactionID)
	}
}

// How often a rotation with nothing to do looks again (and one with moves, through the GCD too).
const ForeverRotationPoll = 50 * time.Millisecond

func (unit *Unit) arcMoveDelay() time.Duration {
	if unit.Type == PetUnit {
		return 0
	}
	if unit.ArcPlay.MoveDelaySet {
		return unit.ArcPlay.MoveDelay
	}
	return ForeverMoveDelay
}

// Where the walk under way ends.
func (unit *Unit) arcWalkTarget() float64 {
	return unit.movementAction.srcPosition + unit.movementAction.moveDistance
}

func (unit *Unit) arcCastBlockedByWalk() bool {
	return ForeverWalkRules && unit.arcWalk.pending
}

func (action *APLActionMove) arcIsReady(sim *Simulation) bool {
	unit := action.unit
	if unit.arcWalk.pending || unit.arcWalk.queued || unit.Hardcast.Expires >= sim.CurrentTime {
		return false
	}
	moveRange := unit.arcSpot(action.moveRange.GetFloat(sim))
	if unit.Moving {
		unit.UpdatePosition(sim, false)
		d := unit.DistanceFromTarget
		return d <= MaxMeleeRange && unit.arcWalkTarget() < d && moveRange > d
	}
	return moveRange != unit.DistanceFromTarget || sim.CurrentTime < 0
}

func (unit *Unit) arcMoveTo(moveRange float64, sim *Simulation, byHand bool) {
	moveRange = unit.arcSpot(moveRange)
	walk := &unit.arcWalk
	if unit.Moving {
		if moveRange == unit.arcWalkTarget() || walk.pending || walk.queued {
			return
		}
		queue := func(sim *Simulation) {
			if unit.Moving {
				walk.queued, walk.queuedRange, walk.queuedByHand = true, moveRange, byHand
			} else {
				unit.moveToNow(moveRange, sim, byHand)
			}
		}
		delay := unit.arcMoveReaction(sim)
		if delay <= 0 || sim.CurrentTime < 0 {
			queue(sim)
			return
		}
		walk.pending, walk.pendingRange = true, moveRange
		unit.logMoveReaction(sim, delay)
		sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
			DoAt: sim.CurrentTime + delay,
			OnAction: func(sim *Simulation) {
				unit.logMoveReactionDone(sim)
				walk.pending = false
				queue(sim)
			},
		}))
		return
	}
	if moveRange == unit.DistanceFromTarget {
		return
	}
	if sim.CurrentTime >= 0 && walk.pending {
		return
	}
	if delay := unit.arcMoveReaction(sim); delay > 0 && sim.CurrentTime >= 0 {
		walk.pending, walk.pendingRange = true, moveRange
		unit.logMoveReaction(sim, delay)
		sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
			DoAt: sim.CurrentTime + delay,
			OnAction: func(sim *Simulation) {
				unit.logMoveReactionDone(sim)
				walk.pending = false
				unit.moveToNow(moveRange, sim, byHand)
				unit.ReactToEvent(sim, false, false)
			},
		}))
		return
	}
	unit.moveToNow(moveRange, sim, byHand)
}

// On arrival: a walk decided on the way in starts now. Else a swing that landed during the walk (so it could not start
// the walk back then) has its decision now, not after a busy GCD.
func (unit *Unit) arcWalkEnded(sim *Simulation) {
	walk := &unit.arcWalk
	swung := walk.swungWhileMoving
	walk.swungWhileMoving = false
	if !walk.queued {
		if swung && unit.Rotation != nil && !sim.Options.Interactive {
			sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
				DoAt:     sim.CurrentTime,
				OnAction: unit.arcReactNow,
			}))
		}
		return
	}
	walk.queued = false
	moveRange, byHand := walk.queuedRange, walk.queuedByHand
	sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
		DoAt: sim.CurrentTime,
		OnAction: func(sim *Simulation) {
			unit.moveToNow(moveRange, sim, byHand)
			unit.ReactToEvent(sim, false, false)
		},
	}))
}

// An auto attack is a decision point: the rotation looks at once, not a reaction time later (a move into melee right
// after the shot, a walk out right after the swing).
func (unit *Unit) arcReactNow(sim *Simulation) {
	if unit.Rotation.HasBlockingControllingAction() {
		unit.Rotation.DoNextAction(sim)
		return
	}
	unit.SetRotationTimer(sim, sim.CurrentTime)
	unit.Rotation.DoNextAction(sim)
}

// A Strict Sequence starts only with the GCD free and every spell in it ready now, not inside the spell queue window:
// queued early, it took control of the rotation and beat a strike placed above it.
func (action *APLActionStrictSequence) arcReadyNow(sim *Simulation) bool {
	if !action.unit.GCD.IsReady(sim) {
		return false
	}
	for _, spell := range action.subactionSpells {
		if !spell.IsReady(sim) {
			return false
		}
	}
	return true
}

// The off hand retries like the main hand: a swing that came ready out of range, with the attack started on the way
// in, waits for the first half-second step after the server sees the unit arrive.
func (aa *AutoAttacks) arcRetryAt(sim *Simulation, ready time.Duration) time.Duration {
	from := max(aa.meleeAttackOnAt, ready)
	seen := sim.CurrentTime + MeleeArrivalDelay
	if seen <= from {
		return from
	}
	return from + (seen-from+MeleeAutoRetryInterval-1)/MeleeAutoRetryInterval*MeleeAutoRetryInterval
}

// Off: a walk out of melee keeps swinging until it is past 5 yd, as this engine does and Arc's engine now does too. On,
// a walk out stops the melee swings as it starts (Arc's engine's old rule), for comparisons.
var ForeverWalkOutStopsMelee = false

func (unit *Unit) arcWalkingOut() bool {
	if !unit.Moving || unit.movementAction == nil || unit.arcWalkTarget() <= MaxMeleeRange || unit.arcWalkTarget() <= unit.DistanceFromTarget {
		return false
	}
	// with yard steps a walk out stops swinging on the yard that reaches the melee edge: walking on, it is past the
	// edge the moment after
	d := unit.DistanceFromTarget
	return ForeverWalkOutStopsMelee || (ForeverYardSteps && d >= MaxMeleeRange && d > unit.movementAction.srcPosition)
}

func (unit *Unit) arcStopMeleeOnWalkOut(sim *Simulation) {
	if unit.arcWalkingOut() && unit.AutoAttacks.mh.enabled {
		unit.AutoAttacks.CancelMeleeSwing(sim)
		unit.AutoAttacks.scheduleMeleeWeaveWakeup(sim)
	}
}

func (unit *Unit) arcResetWalk() {
	unit.arcWalk = arcWalk{}
}
