package rogue

// The Forever rogue rework: Shank and the talents that hang on it, Poised and Composure, Relentless
// Strikes' ranks, Death Mark, Ghostly Strike's stealth-free opener, Blood Splatter and the poisons'
// attack power share.

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// Talent strings in the tree's own order: Assassination 17, Combat 19, Subtlety 18.
const (
	noTalents            = ""
	venomTalent          = "00000000000001"
	relentlessOne        = "0000000000001"
	relentlessFive       = "0000000000005"
	bloodSplatterTalent  = "00000001"
	flashOfSteelFive     = "-000000000000000005"
	composureTwo         = "-0000000002"
	bootKnifeFive        = "--00000000000000005"
	ghostlyStrikeTalent  = "--000000000001"
	deathMarkTalent      = "--000000000000000001"
	deathMarkCoupDeGrace = "-3-000000000000000001"
)

// A rogue with the talents given, the default poisons and one target.
func newRogueSim(t *testing.T, talents string) (*core.Simulation, *Rogue, *core.Unit) {
	t.Helper()
	return newRogueSimWithWeapons(t, talents, 0, 0)
}

// The same with these items in the main and off hand (0 keeps the gear set's).
func newRogueSimWithWeapons(t *testing.T, talents string, mainHand, offHand int32) (*core.Simulation, *Rogue, *core.Unit) {
	t.Helper()

	equipment := core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet
	if mainHand != 0 {
		equipment.Items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand}
	}
	if offHand != 0 {
		equipment.Items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: offHand}
	}
	player := &proto.Player{
		Name:          "Rogue",
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassRogue,
		Equipment:     equipment,
		Consumables:   DefaultConsumables,
		TalentsString: talents,
		Spec:          DefaultOptions,
		Rotation:      &proto.APLRotation{},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: core.LongDuration,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()

	rogue := sim.Raid.Parties[0].Players[0].(RogueAgent).GetRogue()
	return sim, rogue, sim.Encounter.AllTargetUnits[0]
}

func runUntil(t *testing.T, sim *core.Simulation, at time.Duration) {
	t.Helper()
	for sim.CurrentTime < at {
		if sim.Step() {
			t.Fatal("the sim ended early")
		}
	}
}

// Takes Poised off, so a test of something else pays full costs.
func withoutPoised(sim *core.Simulation, rogue *Rogue) {
	rogue.GetAuraByID(core.ActionID{SpellID: poisedRank.ID}).Deactivate(sim)
}

// Casts until one lands, the cooldown cleared before each try.
func castUntilLanded(t *testing.T, sim *core.Simulation, spell *core.Spell, target *core.Unit) {
	t.Helper()
	for i := 0; i < 50; i++ {
		spell.CD.Reset()
		landed := spell.SpellMetrics[target.UnitIndex].Hits + spell.SpellMetrics[target.UnitIndex].Crits
		spell.Unit.GCD.Reset()
		if !spell.Cast(sim, target) {
			t.Fatalf("%v did not cast", spell.ActionID)
		}
		if spell.SpellMetrics[target.UnitIndex].Hits+spell.SpellMetrics[target.UnitIndex].Crits > landed {
			return
		}
	}
	t.Fatalf("%v never landed", spell.ActionID)
}

func TestShankPaysItsEnergyAndFlashOfSteelAddsToIt(t *testing.T) {
	for _, c := range []struct {
		name    string
		talents string
		energy  float64
		cd      time.Duration
	}{
		{"rank 2 Shank", noTalents, 25, 10 * time.Second},
		{"Flash of Steel 5/5", flashOfSteelFive, 40, 8 * time.Second},
	} {
		t.Run(c.name, func(t *testing.T) {
			sim, rogue, target := newRogueSim(t, c.talents)
			withoutPoised(sim, rogue)
			rogue.SpendEnergy(sim, rogue.CurrentEnergy()-50, rogue.relentlessStrikesMetrics)

			if !rogue.Shank.Cast(sim, target) {
				t.Fatal("Shank did not cast")
			}
			if got, want := rogue.CurrentEnergy(), 50+c.energy-rogue.Shank.DefaultCast.Cost; math.Abs(got-want) > 1e-6 {
				t.Errorf("energy after Shank = %.2f, want %.2f", got, want)
			}
			if got := rogue.Shank.CD.Duration; got != c.cd {
				t.Errorf("Shank's cooldown = %v, want %v", got, c.cd)
			}
		})
	}
}

func TestShankNeverMissesNorIsDodged(t *testing.T) {
	sim, rogue, target := newRogueSim(t, noTalents)
	for i := 0; i < 300; i++ {
		rogue.Shank.CD.Reset()
		rogue.GCD.Reset()
		rogue.AddEnergy(sim, 100, rogue.relentlessStrikesMetrics)
		if !rogue.Shank.Cast(sim, target) {
			t.Fatal("Shank did not cast")
		}
	}
	m := rogue.Shank.SpellMetrics[target.UnitIndex]
	if m.Misses+m.Dodges+m.Parries > 0 {
		t.Errorf("Shank missed %d, was dodged %d and parried %d times in 300", m.Misses, m.Dodges, m.Parries)
	}
}

func TestBootKnifeAddsTwoPointsAtFullRank(t *testing.T) {
	sim, rogue, target := newRogueSim(t, bootKnifeFive)
	castUntilLanded(t, sim, rogue.Shank, target)
	if got := rogue.ComboPoints(); got != 2 {
		t.Errorf("combo points after Shank = %d, want 2", got)
	}

	sim, rogue, target = newRogueSim(t, noTalents)
	castUntilLanded(t, sim, rogue.Shank, target)
	if got := rogue.ComboPoints(); got != 0 {
		t.Errorf("combo points after Shank without Boot Knife = %d, want 0", got)
	}
}

func TestShankGivesVenomsPoisonBuff(t *testing.T) {
	sim, rogue, target := newRogueSim(t, venomTalent)
	if rogue.VenomAura == nil || rogue.InstantPoison == nil {
		t.Fatal("no Venom aura or Instant Poison")
	}
	before := rogue.InstantPoison.DirectDamageMultiplier
	castUntilLanded(t, sim, rogue.Shank, target)
	if !rogue.VenomAura.IsActive() {
		t.Fatal("Shank left Venom's buff down")
	}
	if got := rogue.VenomAura.Duration; got != 6*time.Second {
		t.Errorf("Venom lasts %v, want 6s", got)
	}
	after := rogue.InstantPoison.DirectDamageMultiplier
	if got := after / before; math.Abs(got-1.3) > 1e-9 {
		t.Errorf("Instant Poison damage x%.4f under Venom, want x1.3", got)
	}

	_, rogue, _ = newRogueSim(t, noTalents)
	if rogue.VenomAura != nil {
		t.Error("a rogue without Venom has its buff")
	}
}

func TestPoisedCheapensOnlyTheOpener(t *testing.T) {
	for _, c := range []struct {
		name     string
		talents  string
		discount float64
	}{
		{"Poised", noTalents, 10},
		{"Poised with Composure 2/2", composureTwo, 20},
	} {
		t.Run(c.name, func(t *testing.T) {
			sim, rogue, target := newRogueSim(t, c.talents)
			full := float64(rogue.SinisterStrike.Cost.BaseCost)
			if got := rogue.SinisterStrike.Cost.GetCurrentCost(); got != full-c.discount {
				t.Errorf("the opener costs %.0f, want %.0f", got, full-c.discount)
			}
			if !rogue.SinisterStrike.Cast(sim, target) {
				t.Fatal("Sinister Strike did not cast")
			}
			if got := rogue.SinisterStrike.Cost.GetCurrentCost(); got != full {
				t.Errorf("the next one costs %.0f, want the full %.0f", got, full)
			}
		})
	}
}

func TestRelentlessStrikesRanksAndEnergy(t *testing.T) {
	for _, c := range []struct {
		talents  string
		perPoint float64
	}{
		{noTalents, 0},
		{relentlessOne, 0.04},
		{relentlessFive, 0.20},
	} {
		_, rogue, _ := newRogueSim(t, c.talents)
		if got := rogue.relentlessStrikesChancePerPoint; math.Abs(got-c.perPoint) > 1e-9 {
			t.Errorf("%q: chance per point = %.2f, want %.2f", c.talents, got, c.perPoint)
		}
		if got := rogue.relentlessStrikesEnergy; got != 25 {
			t.Errorf("%q: Relentless Strikes restores %.0f energy, want 25", c.talents, got)
		}
	}
}

func deathMarkPayout(t *testing.T, talents string, dealt, bled float64) (paid, cap float64) {
	t.Helper()
	sim, rogue, target := newRogueSim(t, talents)
	if rogue.DeathMark == nil {
		t.Fatal("no Death Mark")
	}
	// Only the damage the test hands the mark gets stored.
	rogue.AutoAttacks.CancelAutoSwing(sim)
	rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	ap := rogue.GetStat(stats.AttackPower)
	crit := rogue.GetStat(stats.PhysicalCritPercent)
	mark := rogue.GetAuraByID(core.ActionID{SpellID: deathMarkRank.ID})
	for i := 0; !mark.IsActive(); i++ {
		if i == 50 {
			t.Fatal("Death Mark never landed")
		}
		rogue.DeathMark.CD.Reset()
		rogue.GCD.Reset()
		rogue.AddEnergy(sim, 100, rogue.relentlessStrikesMetrics)
		if !rogue.DeathMark.Cast(sim, target) {
			t.Fatal("Death Mark did not cast")
		}
	}
	start := sim.CurrentTime

	rogue.OnSpellHitDealt(sim, rogue.SinisterStrike, &core.SpellResult{Target: target, Damage: dealt, Outcome: core.OutcomeHit})
	rogue.OnPeriodicDamageDealt(sim, rogue.Rupture, &core.SpellResult{Target: target, Damage: bled, Outcome: core.OutcomeHit})
	runUntil(t, sim, start+deathMarkRank.Duration()+time.Millisecond)

	payout := rogue.GetSpell(core.ActionID{SpellID: deathMarkRank.ID}.WithTag(deathMarkPayoutTag))
	cap = (156 + 245*5 + 0.04*5*ap) * (1 + crit/100)
	return payout.SpellMetrics[target.UnitIndex].TotalDamage, cap
}

func TestDeathMarkStoresAShareAndPaysItOut(t *testing.T) {
	// 5 points store 60%, Rupture's damage twice that.
	paid, _ := deathMarkPayout(t, deathMarkTalent, 1000, 100)
	if want := 1000*0.6 + 100*1.2; math.Abs(paid-want) > 1e-6 {
		t.Errorf("Death Mark paid %.2f, want %.2f", paid, want)
	}

	paid, cap := deathMarkPayout(t, deathMarkTalent, 1e6, 0)
	if math.Abs(paid-cap) > 1e-6 {
		t.Errorf("Death Mark paid %.2f past its cap of %.2f", paid, cap)
	}

	paid, _ = deathMarkPayout(t, deathMarkCoupDeGrace, 1000, 0)
	if want := 1000 * 0.6 * 1.2; math.Abs(paid-want) > 1e-6 {
		t.Errorf("with Coup de Grace 3/3 Death Mark paid %.2f, want %.2f", paid, want)
	}
}

func TestGhostlyStrikeFreesTheNextOpener(t *testing.T) {
	sim, rogue, target := newRogueSim(t, ghostlyStrikeTalent)
	rogue.BreakStealth(sim)
	rogue.AddEnergy(sim, 100, rogue.relentlessStrikesMetrics)
	if rogue.Ambush.CanCast(sim, target) {
		t.Fatal("Ambush casts out of Stealth before Ghostly Strike")
	}

	castUntilLanded(t, sim, rogue.GhostlyStrike, target)
	rogue.GCD.Reset()
	rogue.AddEnergy(sim, 100, rogue.relentlessStrikesMetrics)
	if !rogue.Ambush.CanCast(sim, target) || !rogue.Garrote.CanCast(sim, target) {
		t.Fatal("Ghostly Strike left Ambush or Garrote needing Stealth")
	}
	if !rogue.Ambush.Cast(sim, target) {
		t.Fatal("Ambush did not cast")
	}
	rogue.GCD.Reset()
	rogue.AddEnergy(sim, 100, rogue.relentlessStrikesMetrics)
	if rogue.Garrote.CanCast(sim, target) {
		t.Error("the opener's buff outlived the Ambush that used it")
	}
}

// The attack power part of a strike's weapon damage, 1400 attack power against none on twin sims, so
// both draw the same weapon roll.
func strikeAttackPowerPart(t *testing.T, talents string, mainHand, offHand int32, strike func(*Rogue, *core.Simulation, float64) float64) float64 {
	t.Helper()
	var parts [2]float64
	for i, ap := range []float64{1400, 0} {
		sim, rogue, _ := newRogueSimWithWeapons(t, talents, mainHand, offHand)
		parts[i] = strike(rogue, sim, ap)
	}
	return parts[0] - parts[1]
}

// Shank states only a weapon percent, so the off hand swings at its own speed: Bagh Nakh's 2.7 s, not
// a one-hander's normalized 2.4 s (the off hand deals half). Ghostly Strike's hotfix gave it a
// normalized weapon damage effect, so Teebu's Blazing Longsword strikes as 2.4 s, not its 2.9 s.
func TestStrikesTakeTheWeaponSpeedTheirDataStates(t *testing.T) {
	const baghNakh, teebus = 250618, 1728
	shank := strikeAttackPowerPart(t, noTalents, 0, baghNakh, func(r *Rogue, sim *core.Simulation, ap float64) float64 {
		return ohStrikeDamage(shankRank, &r.Unit, sim, ap)
	})
	if want := 0.5 * 1400 / 14 * 2.7; math.Abs(shank-want) > 1e-6 {
		t.Errorf("Shank's attack power part %.2f, want %.2f (Bagh Nakh at 2.7 s)", shank, want)
	}
	ghostly := strikeAttackPowerPart(t, ghostlyStrikeTalent, teebus, 0, func(r *Rogue, sim *core.Simulation, ap float64) float64 {
		return mhStrikeDamage(ghostlyStrikeRank, &r.Unit, sim, ap)
	})
	if want := 1400.0 / 14 * 2.4; math.Abs(ghostly-want) > 1e-6 {
		t.Errorf("Ghostly Strike's attack power part %.2f, want %.2f (normalized 2.4 s)", ghostly, want)
	}
}

func TestBloodSplatterLengthensTheBleeds(t *testing.T) {
	for _, c := range []struct {
		talents        string
		rupture, garro time.Duration
	}{
		{noTalents, 0, 0},
		{bloodSplatterTalent, 6 * time.Second, 9 * time.Second},
	} {
		_, rogue, target := newRogueSim(t, c.talents)
		if got := rogue.Rupture.Dot(target).BaseDurationFlat; got != c.rupture {
			t.Errorf("%q: Rupture lasts %v longer, want %v", c.talents, got, c.rupture)
		}
		if got := rogue.Garrote.Dot(target).BaseDurationFlat; got != c.garro {
			t.Errorf("%q: Garrote lasts %v longer, want %v", c.talents, got, c.garro)
		}
	}
}

func TestPoisonsTakeTheNotesAttackPowerShare(t *testing.T) {
	if instantPoisonAPShare != 0.01 || deadlyPoisonAPShare != 0.009 {
		t.Errorf("poison attack power shares %.4f and %.4f, want 0.01 and 0.009", instantPoisonAPShare, deadlyPoisonAPShare)
	}
}
