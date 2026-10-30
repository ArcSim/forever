package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// A hunter of the race with no gear, talents or buffs and a cat up the whole fight, its sim set up
// but not run.
func racialHunter(t *testing.T, race proto.Race) (*core.Simulation, *Hunter) {
	t.Helper()
	player := &proto.Player{
		Name:      "Racial",
		Race:      race,
		Class:     proto.Class_ClassHunter,
		Equipment: &proto.EquipmentSpec{},
		Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{
			ClassOptions: &proto.HunterOptions{PetType: proto.HunterOptions_Cat, PetUptime: 1},
		}}},
		Rotation: &proto.APLRotation{},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

var hunterRaces = []proto.Race{
	proto.Race_RaceHuman, proto.Race_RaceDwarf, proto.Race_RaceOrc, proto.Race_RaceTauren, proto.Race_RaceTroll,
	proto.Race_RaceHighOrderSkyborne, proto.Race_RaceWindshaperSkyborne,
}

// The racial crit and hit each hunter race holds over a Night Elf, whose only crit racial is the
// on-use Elune's Light: Ruthless +1% crit for a Human, Artisan's Precision +1% hit for a Dwarf, and
// nothing for an Orc (Command took Savage's place) or anyone else. The crit is compared at the Night
// Elf's Agility and Intellect, so the races' attributes do not count.
func TestArcRacialCritAndHit(t *testing.T) {
	crit := map[proto.Race]float64{proto.Race_RaceHuman: 1}
	hit := map[proto.Race]float64{proto.Race_RaceDwarf: 1}
	for _, race := range hunterRaces {
		_, hunter := racialHunter(t, race)
		nsim, nelf := racialHunter(t, proto.Race_RaceNightElf)
		nelf.AddStatsDynamic(nsim, stats.Stats{
			stats.Agility:   hunter.GetStat(stats.Agility) - nelf.GetStat(stats.Agility),
			stats.Intellect: hunter.GetStat(stats.Intellect) - nelf.GetStat(stats.Intellect),
		})
		for _, c := range []struct {
			stat stats.Stat
			want float64
		}{
			{stats.PhysicalCritPercent, crit[race]},
			{stats.SpellCritPercent, crit[race]},
			{stats.PhysicalHitPercent, hit[race]},
			{stats.SpellHitPercent, hit[race]},
		} {
			if got := hunter.GetStat(c.stat) - nelf.GetStat(c.stat); math.Abs(got-c.want) > 1e-6 {
				t.Errorf("%v: %s %+.3f over a Night Elf, want %+.0f", race, c.stat.StatName(), got, c.want)
			}
		}
	}
}

// The Human Spirit: a Human's Spirit is its base Spirit and 10%.
func TestArcHumanSpirit(t *testing.T) {
	_, human := racialHunter(t, proto.Race_RaceHuman)
	base := core.BaseStats[core.BaseStatsKey{Race: proto.Race_RaceHuman, Class: proto.Class_ClassHunter}][stats.Spirit]
	if got := human.GetStat(stats.Spirit); math.Abs(got-base*1.10) > 1e-6 {
		t.Errorf("Human Spirit %.2f from a base of %.0f, want %.2f", got, base, base*1.10)
	}
}

// No hunter race carries a critical damage racial: a Tauren's crits hit as hard as a Night Elf's.
func TestArcNoCritDamageRacial(t *testing.T) {
	_, nelf := racialHunter(t, proto.Race_RaceNightElf)
	for _, race := range hunterRaces {
		_, hunter := racialHunter(t, race)
		if got, want := hunter.PseudoStats.CritDamageMultiplier, nelf.PseudoStats.CritDamageMultiplier; got != want {
			t.Errorf("%v: critical damage multiplier x%.4f, a Night Elf's x%.4f", race, got, want)
		}
	}
}

// Command: an Orc's pet deals 5% more damage than a Night Elf's.
func TestArcOrcCommand(t *testing.T) {
	_, orc := racialHunter(t, proto.Race_RaceOrc)
	_, nelf := racialHunter(t, proto.Race_RaceNightElf)
	if got := orc.Pet.PseudoStats.DamageDealtMultiplier / nelf.Pet.PseudoStats.DamageDealtMultiplier; math.Abs(got-1.05) > 1e-9 {
		t.Errorf("Orc pet damage x%.4f over a Night Elf's, want x1.05", got)
	}
}

// Wind Blessed: both Skyborne attack and cast 2% faster than a Night Elf.
func TestArcWindBlessed(t *testing.T) {
	_, nelf := racialHunter(t, proto.Race_RaceNightElf)
	for _, race := range []proto.Race{proto.Race_RaceHighOrderSkyborne, proto.Race_RaceWindshaperSkyborne} {
		_, sky := racialHunter(t, race)
		attack := sky.PseudoStats.AttackSpeedMultiplier / nelf.PseudoStats.AttackSpeedMultiplier
		cast := sky.PseudoStats.CastSpeedMultiplier / nelf.PseudoStats.CastSpeedMultiplier
		if math.Abs(attack-1.02) > 1e-9 || math.Abs(cast-1.02) > 1e-9 {
			t.Errorf("%v: attack speed x%.4f, cast speed x%.4f over a Night Elf; want x1.02 each", race, attack, cast)
		}
	}
}
