package core

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// Realistic weaving reacts on the decision to move, read off the swing timers, not on pressing buttons: the moment a
// walk ends, a button is ready, as in a perfect weave.
func TestArrivalHoldsNoPress(t *testing.T) {
	const button = 910201
	sim, fw, rot := setupFakeRageRotationWith(func(char *Character) {
		char.RegisterSpell(SpellConfig{ActionID: ActionID{SpellID: button}, Flags: SpellFlagAPL, ApplyEffects: func(*Simulation, *Unit, *Spell) {}})
	})
	press := rot.newAPLAction(&proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: ActionID{SpellID: button}.ToProto()}}})
	for _, reaction := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		fw.ArcPlay.WeaveReactionMin, fw.ArcPlay.WeaveReactionSpan = reaction, 100*time.Millisecond
		sim.CurrentTime += time.Second
		fw.arcWalkEnded(sim)
		if !press.IsReady(sim) {
			t.Errorf("with a %v weave reaction a press waits after the walk ends", reaction)
		}
	}
}

// Hawk Time Left asks the hunter's hawks for the hawk picked: the next to leave, the last, or slot 1 or 2; with no
// hawks to ask, 0.
func TestHawkTimeLeftValueReadsTheHawks(t *testing.T) {
	sim, fw, rot := setupFakeRageRotation()
	value := func(pick proto.HawkPick) APLValue {
		return rot.newAPLValue(&proto.APLValue{Value: &proto.APLValue_HawkTimeLeft{HawkTimeLeft: &proto.APLValueHawkTimeLeft{Hawk: pick}}})
	}
	next := value(proto.HawkPick_HawkPickNext)
	if next == nil || next.Type() != proto.APLValueType_ValueTypeDuration {
		t.Fatal("no Hawk Time Left duration value")
	}
	if got := next.GetDuration(sim); got != 0 {
		t.Errorf("with no hawks the value reads %v; want 0", got)
	}
	answers := map[proto.HawkPick]time.Duration{
		proto.HawkPick_HawkPickNext:  3 * time.Second,
		proto.HawkPick_HawkPickLast:  11 * time.Second,
		proto.HawkPick_HawkPickSlot1: 5 * time.Second,
		proto.HawkPick_HawkPickSlot2: 7 * time.Second,
	}
	fw.ArcPlay.HawkTimeLeft = func(_ *Simulation, pick proto.HawkPick) time.Duration { return answers[pick] }
	for pick, want := range answers {
		if got := value(pick).GetDuration(sim); got != want {
			t.Errorf("%s reads %v; want %v", pick, got, want)
		}
	}
}

// The Weave Play value reads the unit's Weaving setting, so a rotation can play each mode its own way.

func TestWeavePlayValueReadsTheSetting(t *testing.T) {
	sim, fw, rot := setupFakeRageRotation()
	value := rot.newAPLValue(&proto.APLValue{Value: &proto.APLValue_WeavePlay{WeavePlay: &proto.APLValueWeavePlay{}}})
	if value == nil {
		t.Fatal("no Weave Play value")
	}
	if value.Type() != proto.APLValueType_ValueTypeInt {
		t.Fatalf("Weave Play is a %s; want an int", value.Type())
	}
	for _, play := range []int32{0, 1, 2} {
		fw.ArcPlay.WeavePlay = play
		if got := value.GetInt(sim); got != play {
			t.Errorf("Weave Play reads %d with the setting at %d", got, play)
		}
	}
}
