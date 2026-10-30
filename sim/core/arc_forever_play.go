package core

import (
	"fmt"
	"math"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// How a player plays the weave and its Auto Shot, set from the class options. The zero value is the site's default:
// the fixed move reaction, a perfect weave, the default run speed.
type ArcPlay struct {
	// The move reaction before each walk, in place of ForeverMoveDelay when MoveDelaySet.
	MoveDelay    time.Duration
	MoveDelaySet bool
	// Realistic weaving: a reaction drawn from WeaveReactionMin to WeaveReactionMin+WeaveReactionSpan before each move
	// (in place of the move reaction), the decision read off the swing timers; buttons are pressed the moment they can
	// be. A walk out to shot range stops WeaveOvershoot yd past the minimum range. Zero is a perfect weave.
	WeaveReactionMin  time.Duration
	WeaveReactionSpan time.Duration
	WeaveOvershoot    float64
	// How long after an ability could first be pressed it is pressed; zero is spamming the button.
	PressReaction time.Duration
	// The Weaving setting itself (0 God Mode, 1 Pro Player, 2 Regular Player), for rotations that read it.
	WeavePlay int32
	// Run speed in yd/s before speed effects; 0 is the default.
	BaseRunSpeed float64
	// The time the picked hawk has left; 0 with no such hawk out. Set by a hunter with Summon Hawk.
	HawkTimeLeft func(sim *Simulation, pick proto.HawkPick) time.Duration
}

func (unit *Unit) arcRealisticWeave() bool {
	return unit.ArcPlay.WeaveReactionMin > 0
}

// A human reaction on a weave step, drawn evenly from its range (measured weaves come out 2.5-2.6 s at best and
// 2.8-3.0 s for most players, the reactions spread over the four steps).
func (unit *Unit) arcWeaveReaction(sim *Simulation) time.Duration {
	return unit.ArcPlay.WeaveReactionMin + time.Duration(sim.RandomFloat("Weave Reaction")*float64(unit.ArcPlay.WeaveReactionSpan))
}

// The wait between deciding to move and the first step.
func (unit *Unit) arcMoveReaction(sim *Simulation) time.Duration {
	if unit.Type != PetUnit && unit.arcRealisticWeave() {
		return unit.arcWeaveReaction(sim)
	}
	return unit.arcMoveDelay()
}

// Where a move stops, in whole yards. With realistic weaving a walk out to shot range ends WeaveOvershoot past the
// minimum range: nobody stops exactly on its edge.
func (unit *Unit) arcSpot(distance float64) float64 {
	spot := math.Round(distance)
	if over := unit.ArcPlay.WeaveOvershoot; over > 0 && spot >= MinRangedRange && spot < MinRangedRange+over {
		spot = math.Round((MinRangedRange+over)*100) / 100
	}
	return spot
}

// A melee swing restarts the Auto Shot timer from this instant: the next shot is a full cycle away, its windup at the
// end of it.
func (aa *AutoAttacks) arcRestartRangedSwing(sim *Simulation) {
	wa := &aa.ranged
	if !aa.AutoSwingRanged || wa.spell == nil {
		return
	}
	wa.updateSwingDuration(wa.unit.TotalRangedHasteMultiplier())
	wa.windupEnd = 0
	wa.swingAt = sim.CurrentTime + wa.curSwingDuration - RangedSwingWindup
	wa.naturalReadyAt = wa.swingAt
	wa.startTimer(sim, sim.CurrentTime+wa.curSwingDuration)
	if wa.enabled {
		sim.rescheduleWeaponAttack(wa.swingAt)
	}
}

// With the timers linked both ways an Auto Shot release restarts the main hand's swing timer too: its next swing comes a
// full swing after the shot, wherever the player stands by then. The off hand keeps its own timer.
func (aa *AutoAttacks) arcRestartMainHandSwing(sim *Simulation) {
	wa := &aa.mh
	if !aa.AutoSwingMelee || wa.spell == nil {
		return
	}
	wa.endRetryWait(sim, "retry")
	wa.updateSwingDuration(wa.unit.TotalMeleeHasteMultiplier())
	wa.swingAt = sim.CurrentTime + wa.curSwingDuration
	wa.naturalReadyAt = wa.swingAt
	wa.startTimer(sim, wa.swingAt)
	if wa.enabled {
		sim.rescheduleWeaponAttack(wa.swingAt)
	}
}

// DelayRangedBy pushes the running Auto Shot cycle back: its windup, or the shot itself while it winds up.
func (aa *AutoAttacks) DelayRangedBy(sim *Simulation, delay time.Duration) {
	wa := &aa.ranged
	if !aa.AutoSwingRanged || wa.spell == nil {
		return
	}
	wa.swingAt += delay
	wa.naturalReadyAt += delay
	if wa.windupEnd != 0 {
		wa.windupEnd += delay
	}
	wa.moveTimer(sim, wa.readyAt+delay)
	if wa.enabled {
		sim.rescheduleWeaponAttack(wa.swingAt)
	}
}

// The Weaving setting (0 God Mode, 1 Pro Player, 2 Regular Player), so a rotation can play differently at a human
// reaction speed.
type APLValueWeavePlay struct {
	DefaultAPLValueImpl
	unit *Unit
}

func (rot *APLRotation) newValueWeavePlay(_ *proto.APLValueWeavePlay, _ *proto.UUID) APLValue {
	return &APLValueWeavePlay{unit: rot.unit}
}
func (value *APLValueWeavePlay) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeInt
}
func (value *APLValueWeavePlay) GetInt(_ *Simulation) int32 {
	return value.unit.ArcPlay.WeavePlay
}
func (value *APLValueWeavePlay) String() string {
	return "Weave Play"
}

// How long a hawk has left: the next to leave, the last, or the one in slot 1 or 2; 0 with no such hawk out.
type APLValueHawkTimeLeft struct {
	DefaultAPLValueImpl
	unit *Unit
	pick proto.HawkPick
}

func (rot *APLRotation) newValueHawkTimeLeft(config *proto.APLValueHawkTimeLeft, _ *proto.UUID) APLValue {
	return &APLValueHawkTimeLeft{unit: rot.unit, pick: config.Hawk}
}
func (value *APLValueHawkTimeLeft) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeDuration
}
func (value *APLValueHawkTimeLeft) GetDuration(sim *Simulation) time.Duration {
	if value.unit.ArcPlay.HawkTimeLeft == nil {
		return 0
	}
	return value.unit.ArcPlay.HawkTimeLeft(sim, value.pick)
}
func (value *APLValueHawkTimeLeft) String() string {
	return fmt.Sprintf("Hawk Time Left(%s)", value.pick)
}
