package core

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

// A Change Target to the unit already targeted is not ready; one to another unit is.
func TestChangeTargetToCurrentTargetIsNotReady(t *testing.T) {
	changeTo := func(index int32) *proto.APLListItem {
		return &proto.APLListItem{Action: &proto.APLAction{Action: &proto.APLAction_ChangeTarget{ChangeTarget: &proto.APLActionChangeTarget{
			NewTarget: &proto.UnitReference{Type: proto.UnitReference_Target, Index: index},
		}}}}
	}
	request := fakeRageSimRequest()
	request.Encounter.Targets = append(request.Encounter.Targets, &proto.Target{Name: "second target", Level: 63, MobType: proto.MobType_MobTypeHumanoid, SwingSpeed: 2})
	sim := setupFakeRageSimFrom(request)
	fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)
	fw.Rotation = fw.newAPLRotation(&proto.APLRotation{
		Type:         proto.APLRotation_TypeAPL,
		PriorityList: []*proto.APLListItem{changeTo(0), changeTo(1)},
	})
	sim.Cleanup()
	sim.Reset()

	first, second := sim.Encounter.AllTargetUnits[0], sim.Encounter.AllTargetUnits[1]
	toFirst, toSecond := fw.Rotation.priorityList[0], fw.Rotation.priorityList[1]
	if fw.CurrentTarget != first {
		t.Fatalf("current target is %q, want %q", fw.CurrentTarget.Label, first.Label)
	}
	if toFirst.IsReady(sim) {
		t.Errorf("Change Target to %q is ready while %q is the current target", first.Label, first.Label)
	}
	if !toSecond.IsReady(sim) {
		t.Fatalf("Change Target to %q is not ready while %q is the current target", second.Label, first.Label)
	}

	toSecond.Execute(sim)
	if fw.CurrentTarget != second {
		t.Fatalf("after Change Target to %q, current target is %q", second.Label, fw.CurrentTarget.Label)
	}
	if !toFirst.IsReady(sim) {
		t.Errorf("Change Target to %q is not ready while %q is the current target", first.Label, second.Label)
	}
}
