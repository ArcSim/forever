package hunter

import (
	"math"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/common/forever"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
	googleproto "google.golang.org/protobuf/proto"
)

// A shipped build with its player and encounter changed, its sim set up for a fight but not run.
func optionSim(t *testing.T, name string, change func(*proto.Player, *proto.Encounter)) (*core.Simulation, *Hunter) {
	t.Helper()
	build := timelineBuild(t, name)
	player := googleproto.Clone(build.Player).(*proto.Player)
	encounter := googleproto.Clone(build.Encounter).(*proto.Encounter)
	change(player, encounter)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{RandomSeed: 20261026},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

var (
	biteHitRe  = regexp.MustCompile(`^\[Target 1\] \{SpellID: ([0-9]+)\} (Hit|Crit) for ([0-9.]+) damage`)
	bleedTicRe = regexp.MustCompile(`^\[Target 1\] \{SpellID: ([0-9]+)\} tick (Hit|Crit) for ([0-9.]+) damage`)
)

// The Lacerating Strikes bleed of a logged fight: each application with the share of the bite that
// started it, per tick, and each tick.
type bleedApplication struct {
	at, fresh float64
}
type bleedTick struct {
	at, amount float64
	crit       bool
}

func bleedLog(t *testing.T, lines []timelineLine) ([]bleedApplication, []bleedTick) {
	t.Helper()
	share := spellData.LaceratingStrikes.FractionAt(1)
	ticks := float64(laceratingStrikesBleed.Duration() / laceratingStrikesBleed.PeriodicEffect().Period())
	bite, bleed := mongooseBiteRank.ID, laceratingStrikesBleed.ID
	var apps []bleedApplication
	var tics []bleedTick
	lastBite := math.NaN()
	for _, l := range lines {
		if m := biteHitRe.FindStringSubmatch(l.text); m != nil && atof(m[1]) == float64(bite) {
			lastBite = atof(m[3])
		}
		if l.text == "Completed cast "+(core.ActionID{SpellID: bleed}).String() {
			apps = append(apps, bleedApplication{l.at, share * lastBite / ticks})
		}
		if m := bleedTicRe.FindStringSubmatch(l.text); m != nil && atof(m[1]) == float64(bleed) {
			tics = append(tics, bleedTick{l.at, atof(m[3]), m[2] == "Crit"})
		}
	}
	if len(apps) < 10 || len(tics) < 20 {
		t.Fatalf("%d bleed applications and %d ticks; want a melee fight's worth", len(apps), len(tics))
	}
	return apps, tics
}

// The ticks after an application (a tick logged at its own time is the old bleed's) up to the next one.
func ticksAfter(apps []bleedApplication, i int, tics []bleedTick) []bleedTick {
	end := math.Inf(1)
	if i+1 < len(apps) {
		end = apps[i+1].at
	}
	var out []bleedTick
	for _, tk := range tics {
		if tk.at > apps[i].at+1e-6 && tk.at <= end+1e-6 {
			out = append(out, tk)
		}
	}
	return out
}

const logStamp = 0.011 // log times carry two decimals

// Lacerating mode Refresh: each bite's bleed replaces the one ticking, so every tick after it is that bite's share
// alone and the first comes a whole tick after it. Strongest: as Refresh, but a weaker bite leaves a stronger bleed
// ticking on its own timer. Rolling (unset): a bite while the bleed ticks adds the unticked damage to the new one and
// the ticks keep their timer.
func TestLaceratingModeOption(t *testing.T) {
	period := laceratingStrikesBleed.PeriodicEffect().Period().Seconds()
	duration := laceratingStrikesBleed.Duration().Seconds()
	fight := func(mode proto.HunterOptions_LaceratingMode) ([]bleedApplication, []bleedTick) {
		return bleedLog(t, playFight(t, "dw_pure_melee", func(o *proto.HunterOptions) { o.LaceratingMode = mode }))
	}

	t.Run("Refresh", func(t *testing.T) {
		apps, tics := fight(proto.HunterOptions_LaceratingRefresh)
		for i, a := range apps {
			after := ticksAfter(apps, i, tics)
			if len(after) > 0 && math.Abs(after[0].at-(a.at+period)) > logStamp {
				t.Errorf("the bleed at %.2f s ticks first at %.2f s; want a whole tick later", a.at, after[0].at)
			}
			for _, tk := range after {
				if !tk.crit && math.Abs(tk.amount-a.fresh) > 0.002 {
					t.Errorf("a tick at %.2f s for %.3f after the bleed at %.2f s; want that bite's %.3f", tk.at, tk.amount, a.at, a.fresh)
				}
			}
		}
	})

	t.Run("Strongest", func(t *testing.T) {
		apps, tics := fight(proto.HunterOptions_LaceratingStrongest)
		base, expires, kept := 0.0, math.Inf(-1), 0
		for i, a := range apps {
			if a.at < expires-1e-6 && a.fresh < base {
				kept++
			} else {
				base, expires = a.fresh, a.at+duration
				if after := ticksAfter(apps, i, tics); len(after) > 0 && math.Abs(after[0].at-(a.at+period)) > logStamp {
					t.Errorf("the bleed at %.2f s replaced the old one but ticks first at %.2f s", a.at, after[0].at)
				}
			}
			for _, tk := range ticksAfter(apps, i, tics) {
				if !tk.crit && math.Abs(tk.amount-base) > 0.002 {
					t.Errorf("a tick at %.2f s for %.3f; want the strongest bleed's %.3f", tk.at, tk.amount, base)
				}
			}
		}
		if kept < 3 {
			t.Errorf("%d weaker bites left a stronger bleed ticking; want a fight's worth", kept)
		}
	})

	t.Run("Rolling", func(t *testing.T) {
		apps, tics := fight(proto.HunterOptions_LaceratingRolling)
		expires, rolled, timerKept := math.Inf(-1), 0, 0
		for i, a := range apps {
			after := ticksAfter(apps, i, tics)
			if a.at < expires-1e-6 && len(after) > 0 {
				rolled++
				if after[0].at < a.at+period-logStamp {
					timerKept++
				}
				for _, tk := range after {
					if !tk.crit && tk.amount <= a.fresh+0.002 {
						t.Errorf("a tick at %.2f s for %.3f after a bite into a ticking bleed at %.2f s; want more than its %.3f", tk.at, tk.amount, a.at, a.fresh)
						break
					}
				}
			}
			if len(after) > 0 {
				expires = after[0].at + (duration - period)
			} else {
				expires = a.at + duration
			}
		}
		if rolled < 5 || timerKept*2 < rolled {
			t.Errorf("%d bites into a ticking bleed, %d of them on its running timer; want a fight's worth, most on it", rolled, timerKept)
		}
	})
}

// Explosive Trap's burn takes its first hit's attack power share, spread over its ticks, only with the option.
func TestExplosiveTrapBurnAPOption(t *testing.T) {
	tick := func(on bool) (damage, apShare float64) {
		sim, hunter := playSim(t, "2h_auto_weave", func(o *proto.HunterOptions) { o.ExplosiveTrapBurnAp = on })
		target := hunter.CurrentTarget
		trap := hunter.GetSpell(core.ActionID{SpellID: explosiveTrapEffectRank.ID})
		if trap == nil {
			t.Fatal("no Explosive Trap")
		}
		trap.SkipCastAndApplyEffects(sim, target)
		trap.AOEDot().TickOnce(sim)
		return trap.SpellMetrics[target.UnitIndex].TotalTickDamage, foreverTrapAP(trap, target)
	}
	off, _ := tick(false)
	on, ap := tick(true)
	area := explosiveTrapEffectRank.EffectN(2)
	ticks := float64(explosiveTrapEffectRank.Duration() / area.Period())
	// the same seed rolls the same outcome, so the ticks differ by their base alone
	want := (area.Average() + ap/ticks) / area.Average()
	if off <= 0 || ap <= 0 || math.Abs(on/off-want) > 1e-9 {
		t.Errorf("a burn tick deals %.3f with the option and %.3f without (ratio %.4f); want ratio %.4f from a %.1f attack power share",
			on, off, on/off, want, ap)
	}
}

// Skysight's Elemental Blessing runs 15% faster and Read Leyline's Energized regenerates mana 50% faster, each
// for 15 s from its cast. Charged and Attuned are removed on entering a raid, so the saved convergence option
// changes nothing: nothing is up before the cast either way.
func TestSkyborneCastIgnoresTheConvergenceOption(t *testing.T) {
	for _, c := range []struct {
		race        proto.Race
		spell, buff int32
		label       string
		want        float64
	}{
		{proto.Race_RaceWindshaperSkyborne, 1259686, 1259688, "Elemental Blessing", 1.15},
		{proto.Race_RaceHighOrderSkyborne, 1259705, 1270842, "Energized", 1.5},
	} {
		for _, on := range []bool{false, true} {
			sim, hunter := optionSim(t, "2h_auto_weave", func(p *proto.Player, _ *proto.Encounter) {
				p.Race = c.race
				p.GetHunter().Options.ClassOptions.SkyborneConvergence = on
			})
			bonus := func() float64 {
				if c.race == proto.Race_RaceWindshaperSkyborne {
					return hunter.PseudoStats.MovementSpeedMultiplier
				}
				return hunter.ManaRegenPerSecondWhileNotCasting()
			}
			aura := hunter.GetAura(c.label)
			if aura == nil || aura.IsActive() {
				t.Errorf("%v convergence %v: %s is missing or up before the cast", c.race, on, c.label)
				continue
			}
			before := bonus()
			hunter.GetSpell(core.ActionID{SpellID: c.spell}).SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
			want := c.want
			if !aura.IsActive() || aura.ActionID.SpellID != c.buff || aura.Duration != 15*time.Second {
				t.Errorf("%v convergence %v: %s up %v as %d for %v; want up as %d for 15s", c.race, on, c.label, aura.IsActive(), aura.ActionID.SpellID, aura.Duration, c.buff)
			}
			if got := bonus() / before; math.Abs(got-want) > 1e-9 {
				t.Errorf("%v convergence %v: %s x%.4f, want x%.2f", c.race, on, c.label, got, want)
			}
		}
	}
}

// A disabled pet ability is not registered, closers included; the rest of the family's stay.
func TestPetDisabledAbilitiesOption(t *testing.T) {
	has := func(spells []*core.Spell, id int32) bool {
		return slices.ContainsFunc(spells, func(s *core.Spell) bool { return s.ActionID.SpellID == id })
	}
	claw, bite, charge := spellData.ClawTriggered.Highest().ID, spellData.BiteTriggered.Highest().ID, spellData.ChargeTriggered.Highest().ID
	for _, c := range []struct {
		pet      proto.HunterOptions_PetType
		disabled []proto.HunterOptions_PetAbility
		want     map[int32]bool
		closers  bool
	}{
		{proto.HunterOptions_Cat, nil, map[int32]bool{claw: true, bite: true}, false},
		{proto.HunterOptions_Cat, []proto.HunterOptions_PetAbility{proto.HunterOptions_PetAbilityClaw}, map[int32]bool{claw: false, bite: true}, false},
		{proto.HunterOptions_Boar, nil, map[int32]bool{charge: true, bite: true}, true},
		{proto.HunterOptions_Boar, []proto.HunterOptions_PetAbility{proto.HunterOptions_PetAbilityCharge}, map[int32]bool{charge: false, bite: true}, true},
	} {
		_, hunter := playSim(t, "bm_weave", func(o *proto.HunterOptions) {
			o.PetType = c.pet
			o.PetDisabledAbilities = c.disabled
		})
		for id, want := range c.want {
			got := has(hunter.Pet.abilities, id)
			if id == charge {
				got = has(hunter.Pet.closers, id)
			}
			if got != want {
				t.Errorf("%v with %v disabled: ability %d registered %v; want %v", c.pet, c.disabled, id, got, want)
			}
		}
	}
}

// Forever pets learn neither Cobra Reflexes nor Pet Aggression: a setting that still carries them, as a
// wowsims import does, leaves the pet as it is without them.
func TestPetIgnoresWowsimsTraining(t *testing.T) {
	pet := func(trained bool) *HunterPet {
		_, hunter := playSim(t, "bm_weave", func(o *proto.HunterOptions) {
			o.CobraReflexes = trained
			o.PetAggression = map[bool]int32{true: 5}[trained]
		})
		return hunter.Pet
	}
	plain, trained := pet(false), pet(true)
	if trained.GetAuraByID(core.ActionID{SpellID: spellData.CobraReflexesTriggered.Highest().ID}) != nil {
		t.Error("the pet carries Cobra Reflexes")
	}
	if got, want := trained.AutoAttacks.MHConfig().DamageMultiplier, plain.AutoAttacks.MHConfig().DamageMultiplier; got != want {
		t.Errorf("the pet's swing damage x%.4f; want x%.4f, as untrained", got, want)
	}
	if trained.GetStats() != plain.GetStats() || trained.PseudoStats != plain.PseudoStats {
		t.Errorf("the pet's stats differ from an untrained pet's:\n%v\n%v", trained.GetStats(), plain.GetStats())
	}
}

// Outside Hyjal a Natural flask gives its Stamina only; in it, the zone bonus too.
func TestOutsideHyjalOption(t *testing.T) {
	flask := forever.NaturalFlasks[1] // Aggression: crit
	sheet := func(outside bool) stats.Stats {
		_, hunter := optionSim(t, "2h_auto_weave", func(p *proto.Player, e *proto.Encounter) {
			p.Consumables.FlaskId = flask
			e.OutsideHyjal = outside
		})
		return hunter.GetStats()
	}
	in, out := sheet(false), sheet(true)
	row := spelldata.MustFind(core.GetConsumableByID(flask).OnUse().BuffId)
	bonus, err := forever.NaturalFlaskZoneBonusOf(row, row.EffectN(2).Average())
	if err != nil {
		t.Fatal(err)
	}
	want := bonus.Stats[stats.PhysicalCritPercent]
	if got := in[stats.PhysicalCritPercent] - out[stats.PhysicalCritPercent]; want <= 0 || math.Abs(got-want) > 1e-9 {
		t.Errorf("the flask's crit in Hyjal over outside: %.3f%%; want the zone bonus %.3f%%", got, want)
	}
	if in[stats.Stamina] != out[stats.Stamina] {
		t.Errorf("Stamina %.1f in Hyjal, %.1f outside; want the same", in[stats.Stamina], out[stats.Stamina])
	}
}

// Every bullet adds its damage per second to the ranged weapon, as the arrows do, and every ammo the option lists
// has a damage.
func TestBulletsAddTheirDamage(t *testing.T) {
	bonus := func(ammo proto.HunterOptions_Ammo) float64 {
		_, hunter := playSim(t, "2h_auto_weave", func(o *proto.HunterOptions) { o.Ammo = ammo })
		return hunter.PseudoStats.BonusRangedDps
	}
	none := bonus(proto.HunterOptions_AmmoNone)
	for name, b := range bullets {
		if id := proto.HunterOptions_Ammo_value[name]; id != b.ItemID {
			t.Errorf("%s is %d in the Ammo option; want its item %d", name, id, b.ItemID)
			continue
		}
		if got := bonus(proto.HunterOptions_Ammo(b.ItemID)) - none; math.Abs(got-b.DPS) > 1e-9 {
			t.Errorf("%s adds %.2f damage per second; want %.2f", b.Name, got, b.DPS)
		}
	}
	for value, name := range proto.HunterOptions_Ammo_name {
		if value != 0 && ammoDPS(name) <= 0 {
			t.Errorf("%s has no damage per second", name)
		}
	}
}
