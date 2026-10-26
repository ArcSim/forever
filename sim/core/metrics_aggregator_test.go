package core_test

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// An action is castable when its unit presses it: a player's APL spell or the hit a queue spell of the
// same action presses, and a pet's ability. Auto attacks, movement and procs are not.
func TestActionMetricsCastableIsWhatTheUnitPresses(t *testing.T) {
	sim, owner := setupPetOwnerSim(&proto.RaidBuffs{}, &proto.PartyBuffs{})
	pet := &owner.pet.Unit

	register := func(unit *core.Unit, id core.ActionID, flags core.SpellFlag) core.ActionID {
		return unit.RegisterSpell(core.SpellConfig{ActionID: id, Flags: flags}).ActionID
	}
	pressed := register(&owner.Unit, core.ActionID{SpellID: 1}, core.SpellFlagAPL)
	queuedHit := register(&owner.Unit, core.ActionID{SpellID: 2}, core.SpellFlagMeleeMetrics)
	register(&owner.Unit, queuedHit.WithTag(1), core.SpellFlagAPL|core.SpellFlagNoMetrics)
	ownerProc := register(&owner.Unit, core.ActionID{SpellID: 3}, core.SpellFlagProc)
	petAbility := register(pet, core.ActionID{SpellID: 4}, core.SpellFlagNone)
	petProc := register(pet, core.ActionID{SpellID: 5}, core.SpellFlagProc)

	sim.Cleanup()

	castable := func(unit *core.Unit, id core.ActionID) bool {
		for _, action := range unit.Metrics.ToProto().Actions {
			if core.ProtoToActionID(action.Id).SameAction(id) {
				return action.IsCastable
			}
		}
		t.Fatalf("%s has no metrics for %s", unit.Label, id)
		return false
	}
	for _, tc := range []struct {
		name string
		unit *core.Unit
		id   core.ActionID
		want bool
	}{
		{"an APL spell", &owner.Unit, pressed, true},
		{"a hit pressed through its tag 1 queue spell", &owner.Unit, queuedHit, true},
		{"the owner's proc", &owner.Unit, ownerProc, false},
		{"the owner's melee", &owner.Unit, owner.AutoAttacks.MHAuto().ActionID, false},
		{"the pet's ability", pet, petAbility, true},
		{"the pet's proc", pet, petProc, false},
		{"the pet's melee", pet, pet.AutoAttacks.MHAuto().ActionID, false},
		{"the pet's movement", pet, core.ActionID{OtherID: proto.OtherAction_OtherActionMove}, false},
	} {
		if got := castable(tc.unit, tc.id); got != tc.want {
			t.Errorf("%s is castable %t, want %t", tc.name, got, tc.want)
		}
	}
}
