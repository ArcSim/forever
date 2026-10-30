package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// A run speed change during a walk covers the rest of the walk at the new speed, with or without yard steps: 3.5 yd at
// 7 yd/s, then 8.5 yd at 9.1 yd/s arrive 1.434 s after the walk starts, not 1.714 s.
func TestWalkSpeedChangeMidWalk(t *testing.T) {
	for _, steps := range []bool{false, true} {
		prev := ForeverYardSteps
		ForeverYardSteps = steps
		sim, fc := setupSwingResetSim(t)
		fc.MoveTo(12, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay())
		if !fc.Moving {
			t.Fatalf("steps %v: the walk did not start after the move reaction", steps)
		}
		walkStart := fc.movementAction.startTime
		sim.AddPendingAction(NewDelayedAction(DelayedActionOptions{
			DoAt:     walkStart + 500*time.Millisecond,
			OnAction: func(sim *Simulation) { fc.MultiplyMovementSpeed(sim, 1.3) },
		}))

		arrival := walkStart + DurationFromSeconds(0.5+8.5/9.1)
		stepSwingSim(t, sim, arrival-10*time.Millisecond)
		if !fc.Moving {
			t.Errorf("steps %v: arrived before %v", steps, arrival)
		}
		stepSwingSim(t, sim, arrival+10*time.Millisecond)
		if fc.Moving || math.Abs(fc.DistanceFromTarget-12) > 1e-9 {
			t.Errorf("steps %v: still walking at %v (%.2f yd), the speed change did not reach the walk", steps, sim.CurrentTime, fc.DistanceFromTarget)
		}
		ForeverYardSteps = prev
	}
}

// The yards the Movement aura logs for the timeline read 5 or less exactly while the unit is in melee range and under
// 11 exactly while it is inside Auto Shot's minimum range, at every event of a walk out and back in.
func TestLoggedYardsFollowTheRangeEdges(t *testing.T) {
	sim, fc := setupSwingResetSim(t)
	for _, dest := range []float64{12, 5} {
		fc.MoveTo(dest, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay())
		if !fc.Moving {
			t.Fatalf("the walk to %v yd did not start after the move reaction", dest)
		}
		for steps := 0; fc.Moving; steps++ {
			if steps > 10000 {
				t.Fatalf("the walk to %v yd did not end", dest)
			}
			d, yards := fc.DistanceFromTarget, fc.moveAura.GetStacks()
			if (yards <= 5) != (d <= MaxMeleeRange) || (yards < 11) != (d < MinRangedRange) {
				t.Fatalf("walking to %v yd: %.3f yd logged as %d at %v", dest, d, yards, sim.CurrentTime)
			}
			sim.Step()
		}
	}
}

// A walk takes exactly its yards at the run speed, 12 yd at 7 yd/s 1.7142857 s, not cut to whole milliseconds.
func TestWalkTimeIsExact(t *testing.T) {
	for _, steps := range []bool{false, true} {
		prev := ForeverYardSteps
		ForeverYardSteps = steps
		sim, fc := setupSwingResetSim(t)
		fc.MoveTo(12, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay())
		if !fc.Moving {
			t.Fatalf("steps %v: the walk did not start after the move reaction", steps)
		}
		arrival := fc.movementAction.startTime + DurationFromSeconds(math.Abs(fc.movementAction.moveDistance)/7)
		stepSwingSim(t, sim, arrival-time.Microsecond)
		if !fc.Moving {
			t.Errorf("steps %v: arrived before %v", steps, arrival)
		}
		stepSwingSim(t, sim, arrival+time.Microsecond)
		if fc.Moving {
			t.Errorf("steps %v: still walking after %v", steps, arrival)
		}
		ForeverYardSteps = prev
	}
}

// Under the walk rules a walk keeps who started it through the move reaction and the queue: a walk by hand is a press
// in the move's first split once it starts, so is one decided during a walk when it starts on arrival, and a walk a
// spell causes lands in the second split with no press.
func TestWalksKeepWhoStartedThemThroughTheReactionAndTheQueue(t *testing.T) {
	sim, fc := setupSwingResetSim(t)
	split := func(i int) SpellMetrics { return fc.moveSpell.splitSpellMetrics[i][fc.CurrentTarget.UnitIndex] }
	fc.WaitUntil(sim, NeverExpires)

	fc.moveTo(12, sim, true)
	stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay())
	if !fc.Moving {
		t.Fatal("the walk by hand did not start after the move reaction")
	}
	fc.moveTo(20, sim, true)
	stepSwingSim(t, sim, sim.CurrentTime+4*time.Second)
	if fc.Moving || math.Abs(fc.DistanceFromTarget-20) > 1e-9 {
		t.Fatalf("the walk decided on the way did not take the unit to 20 yd (%.2f yd, moving %v)", fc.DistanceFromTarget, fc.Moving)
	}
	if got := split(0); got.Casts != 2 || got.ManualCasts != 2 {
		t.Errorf("two walks by hand: %d casts and %d presses by hand, want 2 and 2", got.Casts, got.ManualCasts)
	}

	fc.MoveTo(12, sim)
	stepSwingSim(t, sim, sim.CurrentTime+3*time.Second)
	if got := split(1); got.Casts != 1 || got.ManualCasts != 0 {
		t.Errorf("a walk a spell causes: %d casts and %d presses, want 1 and 0", got.Casts, got.ManualCasts)
	}
}

// Walking in turns the melee attack on as the walk starts, unless the rotation starts the attack itself
// (manual_start_attack): then the walk leaves it off until melee range.
func TestWalkInStartsAttackUnlessTheRotationDoes(t *testing.T) {
	for _, manual := range []bool{false, true} {
		sim, fc := setupSwingResetSim(t)
		fc.MoveTo(12, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+3*time.Second)
		fc.AutoAttacks.meleeAttackOn = false // as an Auto Shot leaves it
		fc.Rotation.manualStartAttack = manual
		fc.MoveTo(0, sim)
		fc.WaitUntil(sim, NeverExpires)
		stepSwingSim(t, sim, sim.CurrentTime+fc.arcMoveDelay()+200*time.Millisecond)
		if fc.DistanceFromTarget <= MaxMeleeRange {
			t.Fatalf("manual %v: already in melee range (%.2f yd)", manual, fc.DistanceFromTarget)
		}
		if on := fc.AutoAttacks.meleeAttackOn; on == manual {
			t.Errorf("manual %v: melee attack on during the walk in = %v", manual, on)
		}
	}
}

// Gear attack power raises ranged attack power once: an item that lists attack power alone (Helm of Narv, +22) gives
// 22 ranged too, and an enchant whose spells already list both (Zandalar Signet of Might, 2606: 30 and 30) is not
// counted twice.
func TestGearAttackPowerRaisesRangedOnce(t *testing.T) {
	stat := func(spec EquipmentSpec) stats.Stats { e := NewEquipmentSet(spec); return e.Stats(0) }
	var helm, shoulder, enchanted EquipmentSpec
	helm[proto.ItemSlot_ItemSlotHead] = ItemSpec{ID: 2245}
	shoulder[proto.ItemSlot_ItemSlotShoulder] = ItemSpec{ID: 2264}
	enchanted[proto.ItemSlot_ItemSlotShoulder] = ItemSpec{ID: 2264, Enchant: 2606}
	if s := stat(helm); s[stats.AttackPower] != 22 || s[stats.RangedAttackPower] != 22 {
		t.Errorf("Helm of Narv: %.0f attack power, %.0f ranged, want 22 and 22", s[stats.AttackPower], s[stats.RangedAttackPower])
	}
	d := stat(enchanted).Subtract(stat(shoulder))
	if d[stats.AttackPower] != 30 || d[stats.RangedAttackPower] != 30 {
		t.Errorf("Zandalar Signet of Might: +%.0f attack power, +%.0f ranged, want 30 and 30", d[stats.AttackPower], d[stats.RangedAttackPower])
	}
}

// Only the consumables the build names are carried: a mana potion that is only in the page's list of usable potions is
// not, so a rotation line for it never fires (Arc Forever Sims carries only the chosen potion).
func TestOnlyTheChosenConsumablesAreCarried(t *testing.T) {
	const chosen, listed, conjured = 990501, 990502, 990503
	addToDatabase(&proto.SimDatabase{Consumables: []*proto.Consumable{
		{Id: chosen, Type: proto.ConsumableType_ConsumableTypePotion, GroupKey: "990/1"},
		{Id: listed, Type: proto.ConsumableType_ConsumableTypePotion, GroupKey: "990/2"},
		{Id: conjured, Type: proto.ConsumableType_ConsumableTypeConjured, GroupKey: "990/3"},
	}})
	got := selectedConsumableIDs(&proto.ConsumesSpec{OffensivePotId: chosen, Potions: []int32{chosen, listed}, ConjuredItems: []int32{conjured}})
	if len(got) != 1 || got[0] != chosen {
		t.Errorf("carried %v, want only the chosen %d", got, chosen)
	}
}
