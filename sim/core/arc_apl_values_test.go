package core

import (
	"strconv"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// Swing Link reads the two link options: 0 none, 1 melee resets Auto Shot, 2 both ways.
func TestSwingLinkValueReadsTheLink(t *testing.T) {
	sim, fw, rot := setupFakeRageRotation()
	value := rot.newAPLValue(&proto.APLValue{Value: &proto.APLValue_SwingLink{SwingLink: &proto.APLValueSwingLink{}}})
	if value == nil || value.Type() != proto.APLValueType_ValueTypeInt {
		t.Fatal("no Swing Link int value")
	}
	for _, tc := range []struct {
		melee, shot bool
		want        int32
	}{{false, false, 0}, {true, false, 1}, {true, true, 2}} {
		fw.AutoAttacks.MeleeResetsAutoShot, fw.AutoAttacks.AutoShotResetsMainHand = tc.melee, tc.shot
		if got := value.GetInt(sim); got != tc.want {
			t.Errorf("melee resets %v, shot resets %v: Swing Link %d, want %d", tc.melee, tc.shot, got, tc.want)
		}
	}
}

// Melee Attack On follows the melee attack: off as an Auto Shot leaves it, on once it starts.
func TestMeleeAttackOnValueFollowsTheAttack(t *testing.T) {
	sim, fc := setupSwingResetSim(t)
	value := fc.Rotation.newAPLValue(&proto.APLValue{Value: &proto.APLValue_MeleeAttackOn{MeleeAttackOn: &proto.APLValueMeleeAttackOn{}}})
	if value == nil || value.Type() != proto.APLValueType_ValueTypeBool {
		t.Fatal("no Melee Attack On bool value")
	}
	fc.AutoAttacks.meleeAttackOn = false
	if value.GetBool(sim) {
		t.Error("Melee Attack On reads true with the attack off")
	}
	fc.AutoAttacks.StartMeleeAttack(sim)
	if !value.GetBool(sim) {
		t.Error("Melee Attack On reads false once the attack starts")
	}
}

// Swing Waiting is how long past ready a weapon's timer sits without a swing; 0 while it runs, and no value for a kind
// that names no one weapon.
func TestSwingWaitingValueReadsPastReady(t *testing.T) {
	sim, fc := setupSwingResetSim(t)
	waiting := func(autoType proto.APLValueAutoAttackType) APLValue {
		return fc.Rotation.newAPLValue(&proto.APLValue{Value: &proto.APLValue_SwingWaiting{SwingWaiting: &proto.APLValueSwingWaiting{AutoType: autoType}}})
	}
	mh, melee := waiting(proto.APLValueAutoAttackType_MainHandAuto), waiting(proto.APLValueAutoAttackType_MeleeAuto)
	if mh == nil || mh.Type() != proto.APLValueType_ValueTypeDuration {
		t.Fatal("no Swing Waiting duration value")
	}
	fc.AutoAttacks.mh.readyAt = sim.CurrentTime - 1500*time.Millisecond
	if got := mh.GetDuration(sim); got != 1500*time.Millisecond {
		t.Errorf("a main hand ready 1.5 s ago waits %v; want 1.5s", got)
	}
	if got := melee.GetDuration(sim); got != 1500*time.Millisecond {
		t.Errorf("Melee reads %v; want the main hand's 1.5s", got)
	}
	fc.AutoAttacks.mh.readyAt = sim.CurrentTime + time.Second
	if got := mh.GetDuration(sim); got != 0 {
		t.Errorf("a main hand ready in 1 s waits %v; want 0", got)
	}
	if waiting(proto.APLValueAutoAttackType_AnyAuto) != nil {
		t.Error("Any names no one weapon, so it makes no value")
	}
}

// Walk Target follows the latest walk decided: during its move reaction, under way, and one decided on the way; with
// no walk, where the unit stands. Walk Pending holds while a decided walk is not under way.
func TestWalkValuesFollowTheWalks(t *testing.T) {
	sim, fc := setupSwingResetSim(t)
	fc.WaitUntil(sim, NeverExpires)
	target := fc.Rotation.newAPLValue(&proto.APLValue{Value: &proto.APLValue_WalkTarget{WalkTarget: &proto.APLValueWalkTarget{}}})
	pending := fc.Rotation.newAPLValue(&proto.APLValue{Value: &proto.APLValue_WalkPending{WalkPending: &proto.APLValueWalkPending{}}})
	if target == nil || target.Type() != proto.APLValueType_ValueTypeFloat || pending == nil || pending.Type() != proto.APLValueType_ValueTypeBool {
		t.Fatal("no Walk Target float or Walk Pending bool value")
	}
	check := func(when string, wantTarget float64, wantPending bool) {
		t.Helper()
		if got := target.GetFloat(sim); got != wantTarget {
			t.Errorf("%s: Walk Target %.2f, want %.2f", when, got, wantTarget)
		}
		if got := pending.GetBool(sim); got != wantPending {
			t.Errorf("%s: Walk Pending %v, want %v", when, got, wantPending)
		}
	}
	start := fc.DistanceFromTarget
	check("standing", start, false)
	fc.moveTo(12, sim, true)
	check("deciding a walk to 12 yd", 12, true)
	stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay())
	check("walking to 12 yd", 12, false)
	fc.moveTo(20, sim, true)
	check("a walk to 20 yd decided on the way", 20, true)
	stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay()+10*time.Millisecond)
	if !fc.arcWalk.queued {
		t.Fatal("the walk to 20 yd is not waiting for the walk to 12 yd to end")
	}
	check("the walk to 20 yd waiting for the arrival at 12 yd", 20, true)
	stepSwingSim(t, sim, sim.CurrentTime+4*time.Second)
	check("arrived at 20 yd", 20, false)
}

// Mana Time To is the engine's own estimate of the regen to the given mana; 0 at or below it, and no value for a unit
// without mana.
func TestManaTimeToTargetValueReadsTheRegen(t *testing.T) {
	sim, fw, rot := setupFakeRageRotation()
	toMana := func(mana float64) APLValue {
		return rot.newAPLValue(&proto.APLValue{Value: &proto.APLValue_ManaTimeToTarget{ManaTimeToTarget: &proto.APLValueManaTimeToTarget{
			TargetMana: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: strconv.FormatFloat(mana, 'f', -1, 64)}}},
		}}})
	}
	if toMana(100) != nil {
		t.Error("a unit with no mana gets a Mana Time To value")
	}

	// a mana pool of 1000 with 2 s of the five-second rule left: 10 mana a second while casting, 30 after
	fw.manaBar.unit = &fw.Unit
	fw.currentMana = 1000
	fw.manaRegenWhileCasting, fw.manaRegenWhileNotCasting = 10, 30
	fw.PseudoStats.FiveSecondRuleRefreshTime = sim.CurrentTime + 2*time.Second
	fw.nextManaTick = sim.CurrentTime + 50*time.Millisecond

	above := toMana(1200)
	if above == nil || above.Type() != proto.APLValueType_ValueTypeDuration {
		t.Fatal("no Mana Time To duration value")
	}
	// 20 mana in the window, the other 180 at 30 a second: 8 s, credited on the next tick after it
	if got, want := above.GetDuration(sim), fw.TimeUntilManaRegen(sim, 1200); got != want || got < 8*time.Second || got > 8100*time.Millisecond {
		t.Errorf("200 mana above the current reads %v; want the regen estimate %v, about 8 s", got, want)
	}
	if got := toMana(999).GetDuration(sim); got != 0 {
		t.Errorf("below the current mana reads %v; want 0", got)
	}
}

// Time Since Last Cast counts from the spell's last cast; never cast this fight, longer than any fight.
func TestSpellTimeSinceLastCastValue(t *testing.T) {
	sim, fw := setupFakeWarlockSim(proto.Race_RaceHuman, &proto.IndividualBuffs{})
	value := fw.Rotation.newAPLValue(&proto.APLValue{Value: &proto.APLValue_SpellTimeSinceLastCast{SpellTimeSinceLastCast: &proto.APLValueSpellTimeSinceLastCast{
		SpellId: fw.Corruption.ActionID.ToProto(),
	}}})
	if value == nil || value.Type() != proto.APLValueType_ValueTypeDuration {
		t.Fatal("no Time Since Last Cast duration value")
	}
	if got := value.GetDuration(sim); got != NeverExpires {
		t.Errorf("never cast reads %v; want longer than any fight", got)
	}
	stepSwingSim(t, sim, time.Second)
	fw.Corruption.Cast(sim, fw.CurrentTarget)
	castAt := sim.CurrentTime
	stepSwingSim(t, sim, castAt+2*time.Second)
	if got := value.GetDuration(sim); got != sim.CurrentTime-castAt || got < 2*time.Second {
		t.Errorf("2 s after the cast reads %v; want %v", got, sim.CurrentTime-castAt)
	}
}
