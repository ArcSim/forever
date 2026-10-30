//go:build with_db

package forever_test

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/common/forever"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

func init() {
	forever.RegisterArcClassicItemEffects2()
}

// A wearer of the items against targets of one mob type: the hunter stand-in, which has a pet, the warrior
// stand-in, which swings its weapons, or the rogue stand-in, which shoots its ranged weapon.
func arcWearer(t *testing.T, class proto.Class, items map[proto.ItemSlot]int32, mobType proto.MobType, targets int) (*core.Simulation, *core.Character) {
	t.Helper()
	specs := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range specs {
		specs[i] = &proto.ItemSpec{}
	}
	for slot, id := range items {
		specs[slot].Id = id
	}
	player := &proto.Player{
		Name: "Wearer", Class: class, Race: proto.Race_RaceOrc,
		Equipment:   &proto.EquipmentSpec{Items: specs},
		Consumables: &proto.ConsumesSpec{},
		Buffs:       &proto.IndividualBuffs{},
		Rotation:    &proto.APLRotation{},
	}
	switch class {
	case proto.Class_ClassHunter:
		core.WithSpec(player, &proto.Player_Warlock{Warlock: &proto.Warlock{}})
	case proto.Class_ClassRogue:
		core.WithSpec(player, &proto.Player_Rogue{Rogue: &proto.Rogue{}})
	default:
		core.WithSpec(player, &proto.Player_DpsWarrior{DpsWarrior: &proto.DpsWarrior{}})
	}
	encounter := &proto.Encounter{Duration: 180}
	for range targets {
		encounter.Targets = append(encounter.Targets, &proto.Target{Name: "target", Level: 60, MobType: mobType})
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid:       &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}},
		Encounter:  encounter,
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].GetCharacter()
}

// Every hit lands, none crits and none is dodged or parried.
func arcSureHits(sim *core.Simulation, character *core.Character) {
	character.AddStatsDynamic(sim, stats.Stats{
		stats.PhysicalHitPercent:  100,
		stats.SpellHitPercent:     100,
		stats.ExpertisePercent:    100,
		stats.PhysicalCritPercent: -character.GetStat(stats.PhysicalCritPercent),
		stats.SpellCritPercent:    -character.GetStat(stats.SpellCritPercent),
	})
}

// The item's proc, checked to roll ppm on the weapon's own speed, and then made to proc on every hit of the
// mask.
func arcForcedProc(t *testing.T, sim *core.Simulation, character *core.Character, label string, itemID int32, ppm float64, rollMask, forceMask core.ProcMask) {
	t.Helper()
	proc := character.GetAura(label + " Proc")
	if proc == nil || proc.Dpm == nil {
		t.Fatalf("%d registers no %q proc on a procs per minute rate", itemID, label)
	}
	want := ppm * core.GetItemByID(itemID).SwingSpeed / 60
	if got := proc.Dpm.Chance(rollMask, sim); want <= 0 || math.Abs(got-want) > 1e-9 {
		t.Errorf("%d's proc rolls %v on a hit, want %v PPM on the weapon's speed, %v", itemID, got, ppm, want)
	}
	*proc.Dpm = *character.NewFixedProcChanceManager(1, forceMask)
}

func rangedShot(unit *core.Unit) *core.Spell {
	return &core.Spell{ProcMask: core.ProcMaskRangedAuto, SpellSchool: core.SpellSchoolPhysical, Unit: unit}
}

// Each item the effects register, and none of the items left out for changing no damage.
func TestArcClassicItemEffectsAreRegistered(t *testing.T) {
	modelled := []int32{
		647, 809, 811, 1263, 2099, 2163, 2291, 2824, 5616, 6831, 9372, 9418, 9465, 9475, 9478, 9511, 10455, 10567,
		10623, 10628, 10761, 10797, 10803, 10805, 11603, 11628, 11629, 11635, 11744, 11809, 11817, 11920, 12243,
		12463, 12469, 12531, 12582, 12583, 12592, 12621, 12709, 12791, 13148, 13171, 13209, 13212, 13213, 13246,
		13285, 13286, 13348, 13361, 13399, 13408, 13505, 13515, 13983, 13984, 14541, 14555, 14576, 15853, 17054,
		17068, 17075, 17730, 17738, 17752, 17780, 18310, 18346, 18410, 18758, 19812, 19991, 19992, 22015, 22061,
		22268, 22321, 23085, 23088, 23089, 23091, 23092, 23093,
	}
	for _, id := range modelled {
		if core.GetItemByID(id) == nil {
			t.Errorf("%d is not in the database", id)
		}
		if !core.HasItemEffect(id) {
			t.Errorf("%d registers no effect", id)
		}
	}
	leftOut := []int32{
		833, 940, 942, 1315, 1490, 8703, 9467, 9477, 10625, 10627, 10847, 11086, 11669, 11808, 11810, 11819, 11832,
		12344, 12532, 12588, 13143, 13198, 13218, 13379, 13382, 13937, 13953, 14022, 14023, 14024, 14557, 15867,
		15873, 17744, 17774, 18345, 18438, 18984, 18986, 20084, 22013, 22060,
	}
	for _, id := range leftOut {
		if core.HasItemEffect(id) {
			t.Errorf("%d registers an effect, want none", id)
		}
	}
}

// A chance-on-hit or equip strike rolls its rate on its weapon's hits and deals its row's roll in its school.
func TestArcClassicStrikesDealTheirRoll(t *testing.T) {
	melee, ranged := proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotRanged
	for _, c := range []struct {
		itemID   int32
		name     string
		slot     proto.ItemSlot
		ppm      float64
		spellID  int32
		school   core.SpellSchool
		min, max float64
	}{
		{811, "Axe of the Deep Woods", melee, 1, 18104, core.SpellSchoolNature, 90, 126},
		{1263, "Brain Hacker", melee, 0.4, 17148, core.SpellSchoolPhysical, 200, 300},
		{2163, "Shadowblade", melee, 1, 18138, core.SpellSchoolShadow, 110, 140},
		{6831, "Black Menace", melee, 1, 13440, core.SpellSchoolShadow, 30, 30},
		{9475, "Diabolic Skiver", melee, 1, 18206, core.SpellSchoolPhysical, 160, 180},
		{9478, "Ripsaw", melee, 1, 16405, core.SpellSchoolPhysical, 75, 75},
		{9511, "Bloodletter Scalpel", melee, 1, 13486, core.SpellSchoolPhysical, 60, 70},
		{10623, "Winter's Bite", melee, 2.2, 13439, core.SpellSchoolFrost, 20, 30},
		{10628, "Deathblow", melee, 1, 16411, core.SpellSchoolPhysical, 160, 160},
		{10761, "Coldrage Dagger", melee, 2.2, 13439, core.SpellSchoolFrost, 20, 30},
		{11603, "Vilerend Slicer", melee, 1, 16405, core.SpellSchoolPhysical, 75, 75},
		{11744, "Bloodfist", melee, 4, 16433, core.SpellSchoolPhysical, 20, 20},
		{11920, "Wraith Scythe", melee, 1, 16414, core.SpellSchoolShadow, 45, 45},
		{12531, "Searing Needle", melee, 1, 16454, core.SpellSchoolFire, 60, 60},
		{12583, "Blackhand Doomsaw", melee, 0.4, 16549, core.SpellSchoolPhysical, 324, 540},
		{13148, "Chillpike", melee, 1, 19260, core.SpellSchoolFrost, 160, 250},
		{13984, "Darrowspike", melee, 1, 18276, core.SpellSchoolFrost, 90, 90},
		{14555, "Alcor's Sunrazor", melee, 1, 18833, core.SpellSchoolFire, 75, 105},
		{17054, "Joonho's Mercy", melee, 1, 20883, core.SpellSchoolArcane, 70, 70},
		{17068, "Deathbringer", melee, 1, 18138, core.SpellSchoolShadow, 110, 140},
		{17075, "Vis'kag the Bloodletter", melee, 0.6, 21140, core.SpellSchoolPhysical, 240, 240},
		{17752, "Satyr's Lash", melee, 1, 18205, core.SpellSchoolShadow, 55, 85},
		{2099, "Dwarven Hand Cannon", ranged, 3, 29639, core.SpellSchoolFire, 33, 49},
		{2824, "Hurricane", ranged, 3, 29502, core.SpellSchoolFrost, 31, 45},
		{10567, "Quillshooter", ranged, 3, 29646, core.SpellSchoolNature, 66, 98},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Only auto attacks are made to proc: a physical strike is itself a melee hit its weapon can proc on.
			class, rollMask, forceMask, hit := proto.Class_ClassWarrior, core.ProcMaskMeleeMHAuto, core.ProcMaskMeleeMHAuto, meleeSwing
			if c.slot == ranged {
				class, rollMask, forceMask, hit = proto.Class_ClassRogue, core.ProcMaskRangedAuto, core.ProcMaskRangedAuto, rangedShot
			}
			sim, character := arcWearer(t, class, map[proto.ItemSlot]int32{c.slot: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
			character.AutoAttacks.CancelAutoSwing(sim)
			arcForcedProc(t, sim, character, c.name, c.itemID, c.ppm, rollMask, forceMask)
			arcSureHits(sim, character)

			strike := character.GetSpell(core.ActionID{SpellID: c.spellID})
			if strike == nil || strike.SpellSchool != c.school {
				t.Fatalf("%d registers %v for %d, want a %v strike", c.itemID, strike, c.spellID, c.school)
			}
			if c.slot == ranged {
				dealHit(t, sim, character, meleeSwing(&character.Unit))
				if got := strike.SpellMetrics[character.CurrentTarget.UnitIndex].Casts; got != 0 {
					t.Errorf("a melee hit cast the ranged proc %d times, want none", got)
				}
			}
			// enough rolls to reach both ends, so a narrower range inside the right one fails too
			lowest, highest := math.Inf(1), math.Inf(-1)
			for i := 1; i <= 200; i++ {
				metrics := &strike.SpellMetrics[character.CurrentTarget.UnitIndex]
				before := metrics.TotalDamage
				dealHit(t, sim, character, hit(&character.Unit))
				if metrics.Casts != int32(i) || metrics.Hits != int32(i) {
					t.Fatalf("hit %d cast the strike %d times for %d hits, want %d of each", i, metrics.Casts, metrics.Hits, i)
				}
				got := metrics.TotalDamage - before
				if got < c.min-1e-9 || got > c.max+1e-9 {
					t.Fatalf("the strike dealt %v, want %v to %v", got, c.min, c.max)
				}
				lowest, highest = min(lowest, got), max(highest, got)
			}
			if span := c.max - c.min; lowest > c.min+0.05*span+1e-9 || highest < c.max-0.05*span-1e-9 {
				t.Errorf("200 strikes rolled %v to %v, want close to both ends of %v to %v", lowest, highest, c.min, c.max)
			}
		})
	}
}

// A damage over time proc ticks its row's amount over its row's duration, after the direct roll some rows
// add; Hookfang Shanker's poison also takes armor from the target while it lasts.
func TestArcClassicDotsTickTheirRow(t *testing.T) {
	for _, c := range []struct {
		itemID   int32
		name     string
		ppm      float64
		spellID  int32
		school   core.SpellSchool
		min, max float64
		tick     float64
		period   time.Duration
		ticks    int32
		armor    float64
	}{
		{809, "Bloodrazor", 1, 17504, core.SpellSchoolPhysical, 0, 0, 12, time.Second * 3, 10, 0},
		{2291, "Kang the Decapitator", 1, 17153, core.SpellSchoolPhysical, 0, 0, 56, time.Second * 3, 10, 0},
		{5616, "Gutwrencher", 1, 16406, core.SpellSchoolPhysical, 0, 0, 8, time.Second * 3, 10, 0},
		{9372, "Sul'thraze the Lasher", 1, 11658, core.SpellSchoolShadow, 90, 210, 25, time.Second * 3, 5, 0},
		{10797, "Firebreather", 1, 16413, core.SpellSchoolFire, 70, 70, 3, time.Second * 2, 3, 0},
		{10803, "Blade of the Wretched", 1, 18088, core.SpellSchoolShadow, 0, 0, 30, time.Second, 3, 0},
		{11635, "Hookfang Shanker", 1, 13526, core.SpellSchoolNature, 0, 0, 7, time.Second * 3, 10, 50},
		{12243, "Smoldering Claw", 1, 15662, core.SpellSchoolFire, 135, 135, 5, time.Second * 2, 3, 0},
		{12463, "Drakefang Butcher", 1, 14118, core.SpellSchoolPhysical, 0, 0, 15, time.Second * 3, 10, 0},
		{12582, "Keris of Zul'Serak", 1, 16528, core.SpellSchoolNature, 0, 0, 10, time.Second * 2, 5, 0},
		{12621, "Demonfork", 1, 16603, core.SpellSchoolShadow, 0, 0, 10, time.Second * 5, 5, 0},
		{12791, "Barman Shanker", 1, 13318, core.SpellSchoolPhysical, 0, 0, 10, time.Second * 3, 10, 0},
		{13285, "The Blackrock Slicer", 1, 17407, core.SpellSchoolPhysical, 50, 150, 6, time.Second, 25, 0},
		{13348, "Demonshear", 1, 17483, core.SpellSchoolShadow, 150, 150, 40, time.Second * 2, 3, 0},
		{13361, "Skullforge Reaver", 1.7, 17484, core.SpellSchoolShadow, 0, 0, 2, time.Second, 30, 0},
		{13399, "Gargoyle Shredder Talons", 1, 18202, core.SpellSchoolPhysical, 0, 0, 11, time.Second * 3, 10, 0},
		{13408, "Soul Breaker", 1, 17506, core.SpellSchoolShadow, 0, 0, 12, time.Second * 3, 10, 0},
		{13983, "Gravestone War Axe", 0.5, 18289, core.SpellSchoolNature, 0, 0, 55, time.Second * 3, 5, 0},
		{14541, "Barovian Family Sword", 0.5, 18652, core.SpellSchoolShadow, 0, 0, 30, time.Second * 3, 5, 0},
		{14576, "Ebon Hilt of Marduk", 1, 18656, core.SpellSchoolShadow, 0, 0, 70, time.Second, 3, 0},
		{15853, "Windreaper", 1, 20586, core.SpellSchoolNature, 0, 0, 10, time.Second * 2, 10, 0},
		{17730, "Gatorbite Axe", 1, 21949, core.SpellSchoolPhysical, 0, 0, 23, time.Second * 3, 10, 0},
		{17738, "Claw of Celebras", 1, 21952, core.SpellSchoolNature, 0, 0, 9, time.Second * 2, 10, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			// An off-hand-only weapon procs on the off hand's swings.
			slot, swing := proto.ItemSlot_ItemSlotMainHand, core.ProcMaskMeleeMHAuto
			if core.GetItemByID(c.itemID).HandType == proto.HandType_HandTypeOffHand {
				slot, swing = proto.ItemSlot_ItemSlotOffHand, core.ProcMaskMeleeOHAuto
			}
			sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{slot: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
			character.AutoAttacks.CancelAutoSwing(sim)
			arcForcedProc(t, sim, character, c.name, c.itemID, c.ppm, swing, swing)
			arcSureHits(sim, character)
			target := character.CurrentTarget
			armor := target.GetStat(stats.Armor)

			spell := character.GetSpell(core.ActionID{SpellID: c.spellID})
			if spell == nil || spell.SpellSchool != c.school {
				t.Fatalf("%d registers %v for %d, want a %v spell", c.itemID, spell, c.spellID, c.school)
			}
			dealHit(t, sim, character, &core.Spell{ProcMask: swing, SpellSchool: core.SpellSchoolPhysical, Unit: &character.Unit})
			*character.GetAura(c.name + " Proc").Dpm = *character.NewFixedProcChanceManager(0, core.ProcMaskMelee)
			dot := spell.Dot(target)
			if !dot.IsActive() || dot.BaseTickCount != c.ticks || dot.BaseTickLength != c.period {
				t.Fatalf("the proc put up %v ticks of %v (active %v), want %d of %v", dot.BaseTickCount, dot.BaseTickLength, dot.IsActive(), c.ticks, c.period)
			}
			if got := armor - target.GetStat(stats.Armor); got != c.armor {
				t.Errorf("the target lost %v armor while it is up, want %v", got, c.armor)
			}
			stepTo(t, sim, sim.CurrentTime+time.Duration(c.ticks)*c.period+time.Millisecond)
			if dot.IsActive() {
				t.Fatal("the damage over time is still up after its duration")
			}
			metrics := spell.SpellMetrics[target.UnitIndex]
			periodic := float64(c.ticks) * c.tick
			if got := metrics.TotalDamage - periodic; got < c.min-1e-6 || got > c.max+1e-6 {
				t.Errorf("the proc dealt %v in all, want %v to %v and %d ticks of %v", metrics.TotalDamage, c.min, c.max, c.ticks, c.tick)
			}
			if got := target.GetStat(stats.Armor); got != armor {
				t.Errorf("the target's armor is %v after it fell off, want %v back", got, armor)
			}
		})
	}
}

// Flame Wrath's ring of fire reaches every enemy.
func TestArcFlameWrathBurnsEveryEnemy(t *testing.T) {
	sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: 11809}, proto.MobType_MobTypeHumanoid, 2)
	character.AutoAttacks.CancelAutoSwing(sim)
	arcForcedProc(t, sim, character, "Flame Wrath", 11809, 1, core.ProcMaskMeleeMHAuto, core.ProcMaskMelee)
	arcSureHits(sim, character)
	dealHit(t, sim, character, meleeSwing(&character.Unit))
	ring := character.GetSpell(core.ActionID{SpellID: 16559})
	for _, target := range sim.Encounter.ActiveTargetUnits {
		if got := ring.SpellMetrics[target.UnitIndex].TotalDamage; got < 130 || got > 170 || ring.SpellSchool != core.SpellSchoolFire {
			t.Errorf("the ring dealt %v %v to %s, want 130 to 170 Fire", got, ring.SpellSchool, target.Label)
		}
	}
}

// The stat procs put their stats up for their duration and take them away after.
func TestArcClassicStatProcs(t *testing.T) {
	for _, c := range []struct {
		itemID   int32
		name     string
		ppm      float64
		bonus    stats.Stats
		duration time.Duration
	}{
		{647, "Destiny", 1, stats.Stats{stats.Strength: 200}, time.Second * 10},
		{9418, "Stoneslayer", 1, stats.Stats{stats.PhysicalDamage: 10}, time.Second * 8},
		{11817, "Lord General's Sword", 1, stats.Stats{stats.AttackPower: 50, stats.RangedAttackPower: 50}, time.Second * 30},
	} {
		t.Run(c.name, func(t *testing.T) {
			sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
			character.AutoAttacks.CancelAutoSwing(sim)
			arcForcedProc(t, sim, character, c.name, c.itemID, c.ppm, core.ProcMaskMeleeMHAuto, core.ProcMaskMelee)
			before := character.GetStatsWithoutDeps()
			dealHit(t, sim, character, meleeSwing(&character.Unit))
			*character.GetAura(c.name + " Proc").Dpm = *character.NewFixedProcChanceManager(0, core.ProcMaskMelee)
			if got := character.GetStatsWithoutDeps().Subtract(before); got != c.bonus {
				t.Errorf("the proc gave %v, want %v", got, c.bonus)
			}
			stepTo(t, sim, sim.CurrentTime+c.duration+time.Millisecond)
			if got := character.GetStatsWithoutDeps(); got != before {
				t.Errorf("after %v the stats are %v, want %v", c.duration, got, before)
			}
		})
	}
}

// Argent Avenger's attack power counts against Undead only, and for melee only.
func TestArcArgentAvengerAgainstUndeadOnly(t *testing.T) {
	for _, c := range []struct {
		mobType proto.MobType
		want    stats.Stats
	}{
		{proto.MobType_MobTypeUndead, stats.Stats{stats.AttackPower: 200}},
		{proto.MobType_MobTypeBeast, stats.Stats{}},
	} {
		sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: 13246}, c.mobType, 1)
		character.AutoAttacks.CancelAutoSwing(sim)
		arcForcedProc(t, sim, character, "Argent Avenger", 13246, 1, core.ProcMaskMeleeMHAuto, core.ProcMaskMelee)
		dealHit(t, sim, character, meleeSwing(&character.Unit))
		if got := bonusAgainstTarget(character); got != c.want {
			t.Errorf("against %v the proc gives %v, want %v", c.mobType, got, c.want)
		}
		stepTo(t, sim, sim.CurrentTime+10*time.Second+time.Millisecond)
		if got := bonusAgainstTarget(character); got != (stats.Stats{}) {
			t.Errorf("against %v after 10 s the bonus is %v, want none", c.mobType, got)
		}
	}
}

// Sprinter's Sword and the Runeblade raise run speed, and only the higher of the two counts.
func TestArcClassicRunSpeed(t *testing.T) {
	_, bare := arcWearer(t, proto.Class_ClassWarrior, nil, proto.MobType_MobTypeHumanoid, 1)
	base := bare.GetMovementSpeed()

	_, runeblade := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: 13505}, proto.MobType_MobTypeHumanoid, 1)
	if got := runeblade.GetMovementSpeed() / base; math.Abs(got-1.08) > 1e-9 {
		t.Errorf("the Runeblade runs %v times as fast, want 1.08", got)
	}

	sim, sprinter := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotMainHand: 18410, proto.ItemSlot_ItemSlotOffHand: 13505}, proto.MobType_MobTypeHumanoid, 1)
	sprinter.AutoAttacks.CancelAutoSwing(sim)
	if got := sprinter.GetMovementSpeed() / base; math.Abs(got-1.08) > 1e-9 {
		t.Errorf("before the proc the wearer runs %v times as fast, want the Runeblade's 1.08", got)
	}
	proc := sprinter.GetAura("Sprinter's Sword Proc")
	if proc == nil || proc.Dpm == nil {
		t.Fatal("18410 registers no proc")
	}
	*proc.Dpm = *sprinter.NewFixedProcChanceManager(1, core.ProcMaskMeleeMH)
	dealHit(t, sim, sprinter, meleeSwing(&sprinter.Unit))
	*proc.Dpm = *sprinter.NewFixedProcChanceManager(0, core.ProcMaskMelee)
	if got := sprinter.GetMovementSpeed() / base; math.Abs(got-1.3) > 1e-9 {
		t.Errorf("under Sprinter's Sword the wearer runs %v times as fast, want 1.3 alone", got)
	}
	stepTo(t, sim, sim.CurrentTime+10*time.Second+time.Millisecond)
	if got := sprinter.GetMovementSpeed() / base; math.Abs(got-1.08) > 1e-9 {
		t.Errorf("after the proc the wearer runs %v times as fast, want 1.08 again", got)
	}
}

// The armor procs: Rivenspike's Puncture Armor stacks three deep, Digmaster 5000's does not stack, and
// Mutilator's Lacerations adds 5 physical damage taken a stack, five deep.
func TestArcClassicDebuffProcs(t *testing.T) {
	for _, c := range []struct {
		itemID int32
		name   string
		ppm    float64
		label  string
		read   func(target *core.Unit) float64
		each   float64
		max    int
	}{
		{13286, "Rivenspike", 2, "Puncture Armor", func(target *core.Unit) float64 { return -target.GetStat(stats.Armor) }, 200, 3},
		{9465, "Digmaster 5000", 2, "Puncture Armor (Digmaster 5000)", func(target *core.Unit) float64 { return -target.GetStat(stats.Armor) }, 100, 1},
		{12469, "Mutilator", 1, "Lacerations", func(target *core.Unit) float64 { return target.PseudoStats.BonusPhysicalDamageTaken }, 5, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
			character.AutoAttacks.CancelAutoSwing(sim)
			arcForcedProc(t, sim, character, c.name, c.itemID, c.ppm, core.ProcMaskMeleeMHAuto, core.ProcMaskMelee)
			arcSureHits(sim, character)
			target := character.CurrentTarget
			base := c.read(target)
			for i := 1; i <= c.max+2; i++ {
				dealHit(t, sim, character, meleeSwing(&character.Unit))
				if got, want := c.read(target)-base, c.each*float64(min(i, c.max)); math.Abs(got-want) > 1e-9 {
					t.Fatalf("after %d procs the debuff is worth %v, want %v", i, got, want)
				}
			}
			if debuff := target.GetAura(c.label); debuff == nil || !debuff.IsActive() {
				t.Fatalf("the target carries no active %q", c.label)
			}
			*character.GetAura(c.name + " Proc").Dpm = *character.NewFixedProcChanceManager(0, core.ProcMaskMelee)
			stepTo(t, sim, sim.CurrentTime+target.GetAura(c.label).Duration+time.Millisecond)
			if got := c.read(target); math.Abs(got-base) > 1e-9 {
				t.Errorf("after the debuff fell off it is worth %v, want none", got-base)
			}
		})
	}
}

// The creature type items give their bonus against their creature types and nothing against others.
func TestArcClassicVersusItems(t *testing.T) {
	ap := func(amount float64) stats.Stats {
		return stats.Stats{stats.AttackPower: amount, stats.RangedAttackPower: amount}
	}
	sd := func(amount float64) stats.Stats { return stats.Stats{stats.SpellDamage: amount} }
	undead, beast, elemental := proto.MobType_MobTypeUndead, proto.MobType_MobTypeBeast, proto.MobType_MobTypeElemental
	for _, c := range []struct {
		itemID  int32
		slot    proto.ItemSlot
		mobType proto.MobType
		want    stats.Stats
	}{
		{10805, proto.ItemSlot_ItemSlotMainHand, undead, ap(30)},
		{11628, proto.ItemSlot_ItemSlotRanged, beast, ap(24)},
		{11629, proto.ItemSlot_ItemSlotRanged, beast, ap(24)},
		{12709, proto.ItemSlot_ItemSlotMainHand, beast, ap(45)},
		{13209, proto.ItemSlot_ItemSlotTrinket1, undead, ap(81)},
		{13212, proto.ItemSlot_ItemSlotNeck, beast, ap(48)},
		{18310, proto.ItemSlot_ItemSlotMainHand, elemental, ap(36)},
		{18758, proto.ItemSlot_ItemSlotMainHand, undead, ap(45)},
		{23088, proto.ItemSlot_ItemSlotChest, undead, ap(81)},
		{23089, proto.ItemSlot_ItemSlotChest, undead, ap(81)},
		{23092, proto.ItemSlot_ItemSlotWrist, undead, ap(45)},
		{23093, proto.ItemSlot_ItemSlotWrist, undead, ap(45)},
		{18346, proto.ItemSlot_ItemSlotLegs, undead, sd(35)},
		{19812, proto.ItemSlot_ItemSlotTrinket1, undead, sd(48)},
		{23085, proto.ItemSlot_ItemSlotChest, undead, sd(48)},
		{23091, proto.ItemSlot_ItemSlotWrist, undead, sd(26)},
	} {
		_, against := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{c.slot: c.itemID}, c.mobType, 1)
		if got := bonusAgainstTarget(against); got != c.want {
			t.Errorf("%d against %v gives %v, want %v", c.itemID, c.mobType, got, c.want)
		}
		_, other := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{c.slot: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
		if got := bonusAgainstTarget(other); got != (stats.Stats{}) {
			t.Errorf("%d against Humanoids gives %v, want none", c.itemID, got)
		}
	}

	// Two items naming the same spell both count.
	_, both := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotMainHand: 18758, proto.ItemSlot_ItemSlotWrist: 23092}, undead, 1)
	if got := bonusAgainstTarget(both); got != ap(90) {
		t.Errorf("Specter's Blade and Wristguards of Undead Slaying give %v together, want %v", got, ap(90))
	}
}

// The stat trinkets go up for their duration on their use's cooldowns.
func TestArcClassicStatTrinkets(t *testing.T) {
	for _, c := range []struct {
		itemID           int32
		bonus            stats.Stats
		duration         time.Duration
		cooldown         time.Duration
		categoryCooldown time.Duration
	}{
		{19991, stats.Stats{stats.AttackPower: 150, stats.RangedAttackPower: 150, stats.PhysicalHitPercent: 2}, 20 * time.Second, 2 * time.Minute, 20 * time.Second},
		{22268, stats.Stats{stats.SpellDamage: 100, stats.HealingPower: 190}, 15 * time.Second, 75 * time.Second, 15 * time.Second},
	} {
		sim, character := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotTrinket1: c.itemID}, proto.MobType_MobTypeHumanoid, 1)
		before := character.GetStatsWithoutDeps()
		spell := useTrinket(t, sim, character, c.itemID, core.CooldownTypeDPS, c.cooldown, 1141, c.categoryCooldown)
		if got := character.GetStatsWithoutDeps().Subtract(before); got != c.bonus {
			t.Errorf("%d gave %v, want %v", c.itemID, got, c.bonus)
		}
		stepTo(t, sim, sim.CurrentTime+c.duration+time.Millisecond)
		if got := character.GetStatsWithoutDeps(); got != before || spell.IsReady(sim) {
			t.Errorf("after %v %d leaves %v (ready %v), want %v and the cooldown running", c.duration, c.itemID, got, spell.IsReady(sim), before)
		}
	}
}

// The damage trinkets: Smolderweb's Eye poisons its target, Smokey's Lighter burns every enemy at once,
// Ramstein's Lightning Bolts strike them a second after the use, and the Chained Essence of Eranikus poisons
// them all after its one-second cast.
func TestArcClassicDamageTrinkets(t *testing.T) {
	for _, c := range []struct {
		itemID           int32
		spellID          int32
		cooldown         time.Duration
		categoryCooldown time.Duration
		after            time.Duration
		landsAt          time.Duration
		total            float64
	}{
		{13213, 13213, 2 * time.Minute, 10 * time.Second, 20 * time.Second, 2 * time.Second, 200},
		{13171, 13171, 5 * time.Minute, 10 * time.Second, time.Millisecond, 0, 125},
		{13515, 17669, 5 * time.Minute, 10 * time.Second, time.Second + time.Millisecond, time.Second, 200},
		{10455, 10455, 15 * time.Minute, 45 * time.Second, 46 * time.Second, 6 * time.Second, 450},
	} {
		sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotTrinket1: c.itemID}, proto.MobType_MobTypeHumanoid, 2)
		character.AutoAttacks.CancelAutoSwing(sim)
		arcSureHits(sim, character)
		start := sim.CurrentTime
		useTrinket(t, sim, character, c.itemID, core.CooldownTypeDPS, c.cooldown, 1141, c.categoryCooldown)
		damage := character.GetSpell(core.ActionID{ItemID: c.itemID})
		if c.spellID != c.itemID {
			damage = character.GetSpell(core.ActionID{SpellID: c.spellID})
		}
		if c.landsAt > 0 {
			stepTo(t, sim, start+c.landsAt-time.Millisecond)
			if got := damage.SpellMetrics[character.CurrentTarget.UnitIndex].TotalDamage; got != 0 {
				t.Errorf("%d dealt %v before %v, want none yet", c.itemID, got, c.landsAt)
			}
		}
		stepTo(t, sim, start+c.after)
		targets := sim.Encounter.ActiveTargetUnits
		if c.itemID == 13213 {
			targets = targets[:1]
		}
		for _, target := range targets {
			if got := damage.SpellMetrics[target.UnitIndex].TotalDamage; math.Abs(got-c.total) > 1e-6 {
				t.Errorf("%d dealt %v to %s, want %v", c.itemID, got, target.Label, c.total)
			}
		}
	}
}

// The Beastmaster pieces reach the pet: the boots raise its damage 3% and the gloves its crit 2%.
func TestArcBeastmasterPiecesReachThePet(t *testing.T) {
	bareSim, bare := arcWearer(t, proto.Class_ClassHunter, nil, proto.MobType_MobTypeHumanoid, 1)
	baseClaw := dealtBy(bareSim, &bare.Pets[0].Unit, stubPetClaw)
	baseCrit := bare.Pets[0].GetStat(stats.PhysicalCritPercent)
	ownerCrit := bare.GetStat(stats.PhysicalCritPercent)

	sim, boots := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotFeet: 22061}, proto.MobType_MobTypeHumanoid, 1)
	if got := dealtBy(sim, &boots.Pets[0].Unit, stubPetClaw); !core.WithinToleranceFloat64(baseClaw*1.03, got, 1e-6) {
		t.Errorf("wearing Beastmaster's Boots, the claw dealt %v, want %v", got, baseClaw*1.03)
	}

	_, gloves := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotHands: 22015}, proto.MobType_MobTypeHumanoid, 1)
	if got := gloves.Pets[0].GetStat(stats.PhysicalCritPercent) - baseCrit; math.Abs(got-2) > 1e-9 {
		t.Errorf("wearing Beastmaster's Gloves, the pet has %v more crit, want 2", got)
	}
	if got := gloves.GetStat(stats.PhysicalCritPercent); got != ownerCrit {
		t.Errorf("wearing Beastmaster's Gloves, the hunter's own crit is %v, want %v unchanged", got, ownerCrit)
	}
}

// Devilsaur Tooth makes the pet's next attack crit, takes the global cooldown, and is spent by the hit.
func TestArcDevilsaurToothCritsThePetsNextAttack(t *testing.T) {
	sim, hunter := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotTrinket1: 19992}, proto.MobType_MobTypeHumanoid, 1)
	pet := hunter.Pets[0]
	baseCrit := pet.GetStat(stats.PhysicalCritPercent)
	useTrinket(t, sim, hunter, 19992, core.CooldownTypeDPS, 2*time.Minute, 1141, 10*time.Second)
	if hunter.GCD.IsReady(sim) {
		t.Error("the use left the global cooldown free, want it running")
	}
	instinct := pet.GetAura("Primal Instinct")
	if instinct == nil || !instinct.IsActive() || pet.GetStat(stats.PhysicalCritPercent)-baseCrit != 100 {
		t.Fatalf("the pet carries %v with %v crit, want Primal Instinct up with 100 more", instinct, pet.GetStat(stats.PhysicalCritPercent))
	}
	dealtBy(sim, &pet.Unit, stubPetClaw)
	if instinct.IsActive() || pet.GetStat(stats.PhysicalCritPercent) != baseCrit {
		t.Errorf("after the claw Primal Instinct is up %v with %v crit, want it spent", instinct.IsActive(), pet.GetStat(stats.PhysicalCritPercent))
	}
}

// Heart of Wyrmthalak procs on melee hits at 0.4 PPM; Blade of Eternal Darkness on one landed damaging spell
// in ten, never on a melee hit.
func TestArcClassicEquipProcs(t *testing.T) {
	sim, heart := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotMainHand: 12583, proto.ItemSlot_ItemSlotTrinket1: 22321}, proto.MobType_MobTypeHumanoid, 1)
	heart.AutoAttacks.CancelAutoSwing(sim)
	proc := heart.GetAura("Heart of Wyrmthalak Proc")
	if proc == nil || proc.Dpm == nil {
		t.Fatal("22321 registers no proc on a procs per minute rate")
	}
	if got, want := proc.Dpm.Chance(core.ProcMaskMeleeMHAuto, sim), 0.4*core.GetItemByID(12583).SwingSpeed/60; math.Abs(got-want) > 1e-9 {
		t.Errorf("Heart of Wyrmthalak rolls %v on a main-hand hit, want 0.4 PPM on the weapon's speed, %v", got, want)
	}
	*proc.Dpm = *heart.NewFixedProcChanceManager(1, core.ProcMaskMelee)
	arcSureHits(sim, heart)
	dealHit(t, sim, heart, meleeSwing(&heart.Unit))
	flameLash := heart.GetSpell(core.ActionID{SpellID: 27655})
	if got := flameLash.SpellMetrics[heart.CurrentTarget.UnitIndex].TotalDamage; got < 120 || got > 180 || flameLash.SpellSchool != core.SpellSchoolFire {
		t.Errorf("Flame Lash dealt %v %v, want 120 to 180 Fire", got, flameLash.SpellSchool)
	}

	sim, blade := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: 17780}, proto.MobType_MobTypeHumanoid, 1)
	blade.AutoAttacks.CancelAutoSwing(sim)
	arcSureHits(sim, blade)
	shadows := blade.GetSpell(core.ActionID{SpellID: 27860})
	for range 200 {
		dealHit(t, sim, blade, meleeSwing(&blade.Unit))
	}
	metrics := &shadows.SpellMetrics[blade.CurrentTarget.UnitIndex]
	if metrics.Casts != 0 {
		t.Errorf("200 melee hits cast Engulfing Shadows %d times, want none", metrics.Casts)
	}
	const spells = 4000
	for range spells {
		dealHit(t, sim, blade, fireSpell(&blade.Unit))
	}
	if rate := float64(metrics.Casts) / spells; math.Abs(rate-0.1) > 0.015 || metrics.TotalDamage != 100*float64(metrics.Hits) {
		t.Errorf("%d spells cast Engulfing Shadows %d times for %v, want one in ten for 100 each", spells, metrics.Casts, metrics.TotalDamage)
	}
}

// Shahram casts each of its six spells about a sixth of the time; of the three that change damage, Fist of
// Shahram hastes melee 30%, Will of Shahram adds 50 to every stat and Flames of Shahram burns every enemy.
func TestArcBlackbladeOfShahram(t *testing.T) {
	sim, character := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotMainHand: 12592}, proto.MobType_MobTypeHumanoid, 2)
	character.AutoAttacks.CancelAutoSwing(sim)
	arcForcedProc(t, sim, character, "Summon Shahram", 12592, 1, core.ProcMaskMeleeMHAuto, core.ProcMaskMelee)
	arcSureHits(sim, character)
	fist, will := character.GetAura("Fist of Shahram"), character.GetAura("Will of Shahram")
	flames := character.GetSpell(core.ActionID{SpellID: 16596})
	if fist == nil || will == nil || flames == nil {
		t.Fatal("12592 registers no Fist, Will or Flames of Shahram")
	}
	var fists, wills int
	const procs = 1200
	for range procs {
		fist.Deactivate(sim)
		will.Deactivate(sim)
		strength := character.GetStatsWithoutDeps()[stats.Strength]
		dealHit(t, sim, character, meleeSwing(&character.Unit))
		if fist.IsActive() {
			fists++
		}
		if will.IsActive() {
			wills++
			if got := character.GetStatsWithoutDeps()[stats.Strength] - strength; got != 50 {
				t.Fatalf("Will of Shahram added %v Strength, want 50", got)
			}
		}
	}
	fist.Deactivate(sim)
	flamesCasts := flames.SpellMetrics[character.CurrentTarget.UnitIndex].Casts
	for name, n := range map[string]int{"Fist": fists, "Will": wills, "Flames": int(flamesCasts)} {
		if rate := float64(n) / procs; math.Abs(rate-1.0/6) > 0.04 {
			t.Errorf("%s of Shahram came up %d times in %d procs, want about a sixth", name, n, procs)
		}
	}
	for _, target := range sim.Encounter.ActiveTargetUnits {
		metrics := flames.SpellMetrics[target.UnitIndex]
		if metrics.Hits == 0 || metrics.TotalDamage < 90*float64(metrics.Hits) || metrics.TotalDamage > 160*float64(metrics.Hits) {
			t.Errorf("Flames of Shahram hit %s %d times for %v, want 90 to 160 a hit", target.Label, metrics.Hits, metrics.TotalDamage)
		}
	}

	before := character.PseudoStats.MeleeSpeedMultiplier
	fist.Activate(sim)
	if got := character.PseudoStats.MeleeSpeedMultiplier / before; math.Abs(got-1.3) > 1e-9 {
		t.Errorf("Fist of Shahram multiplies melee speed by %v, want 1.3", got)
	}
}

// Each set's bonus turns on at its piece count and not one piece before, giving what its bonus states.
func TestArcClassicSetBonuses(t *testing.T) {
	ap := func(amount float64) stats.Stats {
		return stats.Stats{stats.AttackPower: amount, stats.RangedAttackPower: amount}
	}
	head, chest, legs, feet, hands := proto.ItemSlot_ItemSlotHead, proto.ItemSlot_ItemSlotChest, proto.ItemSlot_ItemSlotLegs, proto.ItemSlot_ItemSlotFeet, proto.ItemSlot_ItemSlotHands
	waist, wrist, shoulder := proto.ItemSlot_ItemSlotWaist, proto.ItemSlot_ItemSlotWrist, proto.ItemSlot_ItemSlotShoulder
	type piece struct {
		slot proto.ItemSlot
		id   int32
	}
	for _, c := range []struct {
		name   string
		pieces []piece
		count  int
		bonus  stats.Stats
	}{
		{"The Gladiator", []piece{{head, 11729}, {chest, 11726}, {legs, 11728}}, 3, ap(20)},
		{"The Gladiator", []piece{{head, 11729}, {chest, 11726}, {legs, 11728}, {feet, 11731}, {hands, 11730}}, 5, stats.Stats{stats.PhysicalCritPercent: 1, stats.SpellCritPercent: 1}},
		{"Dal'Rend's Arms", []piece{{proto.ItemSlot_ItemSlotMainHand, 12940}, {proto.ItemSlot_ItemSlotOffHand, 12939}}, 2, ap(50)},
		{"The Postmaster", []piece{{head, 13390}, {chest, 13388}, {legs, 13389}}, 3, stats.Stats{stats.SpellDamage: 23, stats.HealingPower: 23}},
		{"The Postmaster", []piece{{head, 13390}, {chest, 13388}, {legs, 13389}, {feet, 13391}, {proto.ItemSlot_ItemSlotFinger1, 13392}}, 5, stats.Stats{stats.PhysicalHitPercent: 1, stats.SpellHitPercent: 1}},
		{"Cadaverous Garb", []piece{{chest, 14637}, {waist, 14636}, {hands, 14640}}, 3, ap(10)},
		{"Cadaverous Garb", []piece{{chest, 14637}, {waist, 14636}, {hands, 14640}, {legs, 14638}, {feet, 14641}}, 5, stats.Stats{stats.PhysicalHitPercent: 2, stats.SpellHitPercent: 2}},
		{"Necropile Raiment", []piece{{chest, 14626}, {wrist, 14629}}, 2, stats.Stats{stats.HitRating: 5}},
		{"Necropile Raiment", []piece{{chest, 14626}, {wrist, 14629}, {feet, 14631}, {legs, 14632}, {shoulder, 14633}}, 5, stats.Stats{stats.SpellDamage: 23, stats.HealingPower: 23}},
		{"Bloodmail Regalia", []piece{{chest, 14611}, {legs, 14612}}, 2, ap(10)},
		{"Bloodmail Regalia", []piece{{chest, 14611}, {legs, 14612}, {waist, 14614}, {hands, 14615}, {feet, 14616}}, 5, stats.Stats{stats.CritRating: 21}},
		{"Champion's Pursuit", []piece{{chest, 272490}, {hands, 272495}}, 2, stats.Stats{stats.Agility: 20}},
		{"Beastmaster Armor", []piece{{waist, 22010}, {wrist, 22011}, {head, 22013}, {hands, 22015}, {shoulder, 22016}, {legs, 22017}}, 6, ap(40)},
	} {
		t.Run(c.name, func(t *testing.T) {
			label := c.name + " " + string(rune('0'+c.count)) + "P"
			items := map[proto.ItemSlot]int32{}
			for _, p := range c.pieces[:c.count-1] {
				items[p.slot] = p.id
			}
			_, short := arcWearer(t, proto.Class_ClassHunter, items, proto.MobType_MobTypeHumanoid, 1)
			if aura := short.GetAura(label); aura != nil {
				t.Errorf("%d pieces put up %q, want it only at %d", c.count-1, label, c.count)
			}
			items[c.pieces[c.count-1].slot] = c.pieces[c.count-1].id
			sim, full := arcWearer(t, proto.Class_ClassHunter, items, proto.MobType_MobTypeHumanoid, 1)
			aura := full.GetAura(label)
			if aura == nil || !aura.IsActive() {
				t.Fatalf("%d pieces put up no active %q", c.count, label)
			}
			with := full.GetStatsWithoutDeps()
			aura.Deactivate(sim)
			got := with.Subtract(full.GetStatsWithoutDeps())
			for stat, want := range c.bonus {
				if math.Abs(got[stat]-want) > 1e-9 {
					t.Errorf("%q gives %v %v, want %v", label, got[stat], stats.Stat(stat).StatName(), want)
				}
			}
		})
	}
}

// The Postmaster's 2-piece bonus raises run speed 8%; the undead sets' 3-piece bonus multiplies damage
// against Undead by 1.02 and against nothing else.
func TestArcClassicSetMultipliers(t *testing.T) {
	_, bare := arcWearer(t, proto.Class_ClassHunter, nil, proto.MobType_MobTypeHumanoid, 1)
	_, postmaster := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotHead: 13390, proto.ItemSlot_ItemSlotChest: 13388}, proto.MobType_MobTypeHumanoid, 1)
	if got := postmaster.GetMovementSpeed() / bare.GetMovementSpeed(); math.Abs(got-1.08) > 1e-9 {
		t.Errorf("The Postmaster 2P runs %v times as fast, want 1.08", got)
	}
	_, onePiece := arcWearer(t, proto.Class_ClassHunter, map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotHead: 13390}, proto.MobType_MobTypeHumanoid, 1)
	if got := onePiece.GetMovementSpeed() / bare.GetMovementSpeed(); got != 1 {
		t.Errorf("one Postmaster piece runs %v times as fast, want 1", got)
	}

	for _, set := range []struct {
		name string
		ids  [3]int32
	}{
		{"Regalia of Undead Cleansing", [3]int32{23084, 23085, 23091}},
		{"Garb of the Undead Slayer", [3]int32{23082, 23088, 23092}},
		{"Undead Slayer's Armor", [3]int32{23081, 23089, 23093}},
	} {
		items := map[proto.ItemSlot]int32{proto.ItemSlot_ItemSlotHands: set.ids[0], proto.ItemSlot_ItemSlotChest: set.ids[1]}
		for _, c := range []struct {
			pieces  int
			mobType proto.MobType
			want    float64
		}{
			{2, proto.MobType_MobTypeUndead, 1},
			{3, proto.MobType_MobTypeUndead, 1.02},
			{3, proto.MobType_MobTypeDemon, 1},
		} {
			if c.pieces == 3 {
				items[proto.ItemSlot_ItemSlotWrist] = set.ids[2]
			}
			_, character := arcWearer(t, proto.Class_ClassHunter, items, c.mobType, 1)
			if got := character.AttackTables[character.CurrentTarget.UnitIndex].DamageDealtMultiplier; math.Abs(got-c.want) > 1e-9 {
				t.Errorf("%d pieces of %s deal %v times the damage to %v, want %v", c.pieces, set.name, got, c.mobType, c.want)
			}
		}
	}
}

// The Gladiator's 4-piece Roar of the Crowd and Bloodmail Regalia's 3-piece bleed proc at 1 PPM off melee
// hits: Roar of the Crowd hastes melee 5% for 10 s, and the bleed ticks 7 a second for 10 s.
func TestArcClassicSetProcs(t *testing.T) {
	sim, gladiator := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotHead: 11729, proto.ItemSlot_ItemSlotChest: 11726, proto.ItemSlot_ItemSlotLegs: 11728, proto.ItemSlot_ItemSlotFeet: 11731,
	}, proto.MobType_MobTypeHumanoid, 1)
	gladiator.AutoAttacks.CancelAutoSwing(sim)
	arcSureHits(sim, gladiator)
	roar := gladiator.GetAura("Roar of the Crowd")
	if roar == nil {
		t.Fatal("The Gladiator 4P registers no Roar of the Crowd")
	}
	before := gladiator.PseudoStats.MeleeSpeedMultiplier
	swings := 0
	for !roar.IsActive() {
		if swings++; swings > 2000 {
			t.Fatal("2000 melee hits brought no Roar of the Crowd")
		}
		dealHit(t, sim, gladiator, meleeSwing(&gladiator.Unit))
	}
	if got := gladiator.PseudoStats.MeleeSpeedMultiplier / before; math.Abs(got-1.05) > 1e-9 || roar.Duration != 10*time.Second {
		t.Errorf("Roar of the Crowd multiplies melee speed by %v for %v, want 1.05 for 10s", got, roar.Duration)
	}

	sim, bloodmail := arcWearer(t, proto.Class_ClassWarrior, map[proto.ItemSlot]int32{
		proto.ItemSlot_ItemSlotChest: 14611, proto.ItemSlot_ItemSlotLegs: 14612, proto.ItemSlot_ItemSlotWaist: 14614,
	}, proto.MobType_MobTypeHumanoid, 1)
	bloodmail.AutoAttacks.CancelAutoSwing(sim)
	arcSureHits(sim, bloodmail)
	bleed := bloodmail.GetSpell(core.ActionID{SpellID: 1299739})
	if bleed == nil {
		t.Fatal("Bloodmail Regalia 3P registers no bleed")
	}
	dot := bleed.Dot(bloodmail.CurrentTarget)
	swings = 0
	for !dot.IsActive() {
		if swings++; swings > 2000 {
			t.Fatal("2000 melee hits brought no bleed")
		}
		dealHit(t, sim, bloodmail, meleeSwing(&bloodmail.Unit))
	}
	if dot.BaseTickCount != 10 || dot.BaseTickLength != time.Second {
		t.Errorf("the bleed runs %d ticks of %v, want 10 of 1s", dot.BaseTickCount, dot.BaseTickLength)
	}
	metrics := &bleed.SpellMetrics[bloodmail.CurrentTarget.UnitIndex]
	stepTo(t, sim, sim.CurrentTime+10*time.Second+time.Millisecond)
	if metrics.Ticks != 10 || math.Abs(metrics.TotalDamage-70) > 1e-6 {
		t.Errorf("the bleed ticked %d times for %v, want 10 ticks of 7", metrics.Ticks, metrics.TotalDamage)
	}
}
