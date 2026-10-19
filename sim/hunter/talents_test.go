package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
	googleProto "google.golang.org/protobuf/proto"
)

// A reset sim of one hunter with the talents and, unless PetNone, a pet of that family up the whole fight.
func newTalentSim(t *testing.T, talents string, pet proto.HunterOptions_PetType) (*core.Simulation, *Hunter) {
	t.Helper()
	spec := &proto.Player_Hunter{Hunter: googleProto.Clone(DefaultOptions.Hunter).(*proto.Hunter)}
	spec.Hunter.Options.ClassOptions.PetType = pet
	spec.Hunter.Options.ClassOptions.PetUptime = 1
	player := &proto.Player{
		Name:          "Talents",
		Race:          proto.Race_RaceOrc,
		Class:         proto.Class_ClassHunter,
		Equipment:     &proto.EquipmentSpec{},
		TalentsString: talents,
		Spec:          spec,
		Rotation:      &proto.APLRotation{},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

func assertSpeed(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s moves at %.4f yd/s, want %.4f", what, got, want)
	}
}

// Aspect of the Cheetah's 30% with Pathfinding's 6% on top, and Dazed's 70% slow on what is left.
func TestAspectOfTheCheetahSpeedWithPathfindingAndDazed(t *testing.T) {
	for _, tc := range []struct {
		talents string
		speed   float64
	}{
		{"", 0.30},
		{"00002", 0.36},
	} {
		sim, hunter := newTalentSim(t, tc.talents, proto.HunterOptions_PetNone)
		base := hunter.GetMovementSpeed()

		cheetah := hunter.GetSpell(core.ActionID{SpellID: aspectOfTheCheetahRank.ID})
		if cheetah == nil {
			t.Fatal("the hunter has no Aspect of the Cheetah")
		}
		cheetah.SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
		assertSpeed(t, "a hunter in Cheetah with talents "+tc.talents, hunter.GetMovementSpeed(), base*(1+tc.speed))

		dazed := hunter.GetAura(dazedRank.Name)
		dazed.Activate(sim)
		assertSpeed(t, "a dazed hunter in Cheetah", hunter.GetMovementSpeed(), base*(1+tc.speed)*0.3)
		dazed.Deactivate(sim)

		hunter.GetSpell(core.ActionID{SpellID: aspectOfTheHawkRank.ID}).SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
		if hunter.AspectOfTheCheetahAura.IsActive() {
			t.Error("Aspect of the Hawk left Aspect of the Cheetah up, want one aspect at a time")
		}
		assertSpeed(t, "a hunter back in Hawk", hunter.GetMovementSpeed(), base)
	}
}

// Survival Tactics' 10% hit on the two damage traps, on the effect spells that roll the hit.
func TestSurvivalTacticsRaisesTrapHit(t *testing.T) {
	for _, tc := range []struct {
		talents string
		hit     float64
	}{
		{"", 0},
		{"--0000000002", 10},
	} {
		_, hunter := newTalentSim(t, tc.talents, proto.HunterOptions_PetNone)
		for _, rank := range []int32{immolationTrapEffectRank.ID, explosiveTrapEffectRank.ID} {
			spell := hunter.GetSpell(core.ActionID{SpellID: rank})
			if spell == nil {
				t.Fatalf("the hunter has no trap effect %d", rank)
			}
			if spell.BonusHitPercent != tc.hit {
				t.Errorf("trap effect %d with talents %q has %v%% bonus hit, want %v%%", rank, tc.talents, spell.BonusHitPercent, tc.hit)
			}
		}
	}
}

// Bestial Swiftness' 30% on the pet, which Dash's larger speed replaces while it is up.
func TestBestialSwiftnessSpeedsThePetUnderDash(t *testing.T) {
	sim, hunter := newTalentSim(t, "000000100", proto.HunterOptions_Wolf)
	pet := hunter.Pet
	assertSpeed(t, "a Bestial Swiftness pet", pet.GetMovementSpeed(), 8*1.3)

	dash := pet.closers[0]
	if dash.ActionID.SpellID != spellData.DashTriggered.Highest().ID {
		t.Fatalf("the wolf's closer is %v, want Dash", dash.ActionID)
	}
	dash.SkipCastAndApplyEffects(sim, pet.CurrentTarget)
	assertSpeed(t, "a dashing Bestial Swiftness pet", pet.GetMovementSpeed(), 8*(1+spellData.DashTriggered.Highest().EffectN(1).Percent()))
}

// Intimidation's 100% crit stays on the pet until an attack of its lands.
func TestIntimidationCritsThePetsNextLandedAttack(t *testing.T) {
	sim, hunter := newTalentSim(t, "0000000000001", proto.HunterOptions_Wolf)
	pet := hunter.Pet
	crit := pet.GetStats()[stats.PhysicalCritPercent]

	intimidation := hunter.GetSpell(core.ActionID{SpellID: intimidationRank.ID})
	if intimidation == nil {
		t.Fatal("the hunter has no Intimidation")
	}
	intimidation.SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
	aura := pet.GetAura(intimidationRank.Name)
	if got := pet.GetStats()[stats.PhysicalCritPercent]; math.Abs(got-crit-100) > 1e-9 {
		t.Errorf("the intimidating pet has %v%% crit, want %v%%", got, crit+100)
	}

	swing := pet.AutoAttacks.MHAuto()
	aura.OnSpellHitDealt(aura, sim, swing, &core.SpellResult{Target: pet.CurrentTarget, Outcome: core.OutcomeMiss})
	if !aura.IsActive() {
		t.Error("a missed swing spent Intimidation, want it kept for an attack that lands")
	}
	aura.OnSpellHitDealt(aura, sim, swing, &core.SpellResult{Target: pet.CurrentTarget, Outcome: core.OutcomeCrit})
	if aura.IsActive() {
		t.Error("a landed swing left Intimidation up, want its one charge spent")
	}
	if got := pet.GetStats()[stats.PhysicalCritPercent]; math.Abs(got-crit) > 1e-9 {
		t.Errorf("the pet has %v%% crit after Intimidation, want %v%%", got, crit)
	}
}
