package core

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// With a press reaction a button goes that long after it could first be pressed: a spell on the GCD cast over and over
// goes every GCD plus the reaction, the first one the reaction into the fight; with none, at every GCD's end. A move
// the rotation never takes keeps it looking through the GCD, as a weave rotation does.
func TestPressReactionWaitsAfterTheButtonIsReady(t *testing.T) {
	const button = 910301
	for _, reaction := range []time.Duration{0, 230 * time.Millisecond} {
		sim := setupFakeRageSimFrom(fakeRageSimRequest())
		fw := sim.Raid.Parties[0].Players[0].(*FakeRageWarrior)
		var casts []time.Duration
		sim.Environment.State = Constructed
		spell := fw.RegisterSpell(SpellConfig{
			ActionID: ActionID{SpellID: button},
			Flags:    SpellFlagAPL,
			Cast:     CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				casts = append(casts, sim.CurrentTime)
			},
		})
		sim.Environment.State = Finalized
		spell.finalize()
		fw.Rotation = fw.newAPLRotation(&proto.APLRotation{
			Type: proto.APLRotation_TypeAPL,
			PriorityList: []*proto.APLListItem{
				{Action: &proto.APLAction{Condition: constValue("false"), Action: &proto.APLAction_Move{Move: &proto.APLActionMove{RangeFromTarget: constValue("20")}}}},
				{Action: &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: ActionID{SpellID: button}.ToProto()}}}},
			},
		})
		fw.ArcPlay.PressReaction = reaction
		// two fights: nothing the first one saw ready carries into the next
		for fight := 1; fight <= 2; fight++ {
			casts = nil
			sim.Cleanup()
			sim.Reset()
			sim.PrePull()
			sim.Step()
			fw.AutoAttacks.CancelAutoSwing(sim)
			stepSwingSim(t, sim, 7*time.Second)

			if len(casts) < 4 {
				t.Fatalf("a %v reaction, fight %d: %d casts by 7 s", reaction, fight, len(casts))
			}
			if casts[0] != reaction {
				t.Errorf("a %v reaction, fight %d: the first cast at %v; want %v", reaction, fight, casts[0], reaction)
			}
			for i := 1; i < len(casts); i++ {
				if gap := casts[i] - casts[i-1]; (gap - GCDDefault - reaction).Abs() > time.Millisecond {
					t.Errorf("a %v reaction, fight %d: casts %v apart at %v; want %v", reaction, fight, gap, casts[i], GCDDefault+reaction)
				}
			}
		}
	}
}
