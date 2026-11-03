package hunter

import (
	"math"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
	"google.golang.org/protobuf/encoding/protojson"
)

// A hunter with these talents and rotation, no gear, its sim set up and reset.
func dataPassHunter(t *testing.T, talents string, rotation *proto.APLRotation) (*core.Simulation, *Hunter) {
	t.Helper()
	player := &proto.Player{
		Name:          "Data pass",
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassHunter,
		Equipment:     &proto.EquipmentSpec{},
		TalentsString: talents,
		Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{
			ClassOptions: &proto.HunterOptions{PetType: proto.HunterOptions_Cat, PetUptime: 1},
		}}},
		Rotation: rotation,
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

// Trueshot Aura gives its top rank's 100 ranged attack power (the ranks are 20 / 35 / 50 / 75 / 100).
func TestTrueshotAuraGivesItsTopRanksHundred(t *testing.T) {
	if got := buffs.TrueshotAuraValue(0); got != 100 {
		t.Errorf("Trueshot Aura gives %v ranged attack power, want 100", got)
	}
}

// Aspect of the Hawk rank 7 and Serpent Sting rank 9 come only from Ruins of Ahn'Qiraj books: the hunter knows the
// trainer's rank 6 (+110) and rank 8 (100 a tick).
func TestTrainerRanks(t *testing.T) {
	if aspectOfTheHawkRank.ID != 14322 || aspectOfTheHawkRank.EffectN(1).Average() != 110 {
		t.Errorf("Aspect of the Hawk is %d (+%v), want rank 6 (14322, +110)", aspectOfTheHawkRank.ID, aspectOfTheHawkRank.EffectN(1).Average())
	}
	if serpentStingRank.ID != 13555 || foreverSerpentStingTickBase != 100 {
		t.Errorf("Serpent Sting is %d (%v a tick), want rank 8 (13555, 100)", serpentStingRank.ID, foreverSerpentStingTickBase)
	}
	_, hunter := dataPassHunter(t, "", &proto.APLRotation{})
	for _, id := range []int32{25295, 25296} {
		if hunter.GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("the hunter knows %d, a Ruins of Ahn'Qiraj book rank", id)
		}
	}
	if hunter.GetSpell(core.ActionID{SpellID: 13555}) == nil {
		t.Error("the hunter does not know Serpent Sting rank 8 (13555)")
	}
}

// A rotation saved with the book ranks gets the trainer ranks when the hunter is built, so it still casts.
func TestRotationBookRanksMigrate(t *testing.T) {
	rot := &proto.APLRotation{}
	const aplJSON = `{"type":"TypeAPL","priorityList":[{"action":{"condition":{"cmp":{"op":"OpLe","lhs":{"dotRemainingTime":` +
		`{"spellId":{"spellId":25295,"rank":9}}},"rhs":{"const":{"val":"1s"}}}},"castSpell":{"spellId":{"spellId":25295,"rank":9}}}},` +
		`{"action":{"castSpell":{"spellId":{"spellId":25296,"rank":7}}}}]}`
	if err := protojson.Unmarshal([]byte(aplJSON), rot); err != nil {
		t.Fatalf("rotation: %v", err)
	}
	dataPassHunter(t, "", rot)
	got, _ := protojson.Marshal(rot)
	for _, old := range []string{"25295", "25296"} {
		if strings.Contains(string(got), old) {
			t.Errorf("the rotation still names %s once the hunter is built: %s", old, got)
		}
	}
}

// Efficiency lowers Strider Kick's cost too (the launch realms' Efficiency names Strider Kick's spell family flag).
func TestEfficiencyCoversStriderKick(t *testing.T) {
	cost := func(talents string) float64 {
		_, hunter := dataPassHunter(t, talents, &proto.APLRotation{})
		kick := hunter.GetSpell(core.ActionID{SpellID: striderKickRank.ID})
		if kick == nil || kick.Cost == nil {
			t.Fatal("no Strider Kick")
		}
		return kick.Cost.PercentModifier
	}
	// Efficiency 0 / 5, Resourcefulness 2/2 and Strider Kick in both
	none, five := cost("-00530500101-500230230250222151"), cost("-00535500101-500230230250222151")
	if math.Abs(five/none-0.85) > 1e-9 {
		t.Errorf("Efficiency 5/5 leaves Strider Kick %.4f of its cost, want 0.85", five/none)
	}
}

// A Serpent Sting tick is the rank's base plus 5% of ranged attack power, read when the sting lands.
func TestSerpentStingTickTakesFivePercentOfRangedAttackPower(t *testing.T) {
	sim, hunter := dataPassHunter(t, "", &proto.APLRotation{})
	dot := hunter.GetSpell(core.ActionID{SpellID: serpentStingRank.ID}).Dot(hunter.CurrentTarget)
	tick := func() float64 {
		dot.TakeSnapshot(sim)
		return dot.SnapshotBaseDamage
	}
	before := tick()
	hunter.AddStatDynamic(sim, stats.RangedAttackPower, 1000)
	if got := tick() - before; math.Abs(got-50) > 1e-9 {
		t.Errorf("1000 more ranged attack power adds %v to a tick, want 50", got)
	}
}

// The rows state no attack power share for Arcane Shot; the Forever notes set 15% of ranged attack power.
func TestArcaneShotTakesFifteenPercentOfRangedAttackPower(t *testing.T) {
	if arcaneShotAPShare != 0.15 {
		t.Errorf("Arcane Shot takes %v of ranged attack power, want 0.15", arcaneShotAPShare)
	}
}
