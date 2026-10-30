package forever

import (
	"time"

	"github.com/wowsims/forever/sim/common/itemhelpers"
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// Effects of the classic dungeon and world items the Forever client does not carry. Their spells are not
// in the spell store either, so the amounts are the Classic client's rows. Where the client states no
// proc rate, a chance-on-hit weapon procs at 1 PPM, or at the known rate of an item with the same proc.
func RegisterArcClassicItemEffects2() {
	registerArcClassicStrikes()
	registerArcClassicDots()
	registerArcClassicBuffProcs()
	registerArcClassicDebuffProcs()
	registerArcClassicVersusItems()
	registerArcClassicTrinkets()
	registerArcClassicPetItems()
}

// A chance-on-hit strike for a roll of minDmg to maxDmg.
func arcClassicStrike(itemID int32, name string, ppm float64, spellID int32, school core.SpellSchool, defense core.DefenseType, minDmg, maxDmg float64) {
	itemhelpers.CreateWeaponCoHProcDamage(itemhelpers.WeaponProcDamage{
		ItemID:      itemID,
		Name:        name,
		PPM:         ppm,
		SpellID:     spellID,
		School:      school,
		DefenseType: defense,
		MinDmg:      minDmg,
		MaxDmg:      maxDmg,
	})
}

func registerArcClassicStrikes() {
	arcClassicStrike(811, "Axe of the Deep Woods", 1, 18104, core.SpellSchoolNature, core.DefenseTypeMagic, 90, 126)
	// The Lobotomizer's rate for the same strike. The Intellect it takes from the target changes no damage.
	arcClassicStrike(1263, "Brain Hacker", 0.4, 17148, core.SpellSchoolPhysical, core.DefenseTypeMelee, 200, 300)
	arcClassicStrike(2163, "Shadowblade", 1, 18138, core.SpellSchoolShadow, core.DefenseTypeMagic, 110, 140)
	arcClassicStrike(6831, "Black Menace", 1, 13440, core.SpellSchoolShadow, core.DefenseTypeMagic, 30, 30)
	arcClassicStrike(9475, "Diabolic Skiver", 1, 18206, core.SpellSchoolPhysical, core.DefenseTypeMelee, 160, 180)
	arcClassicStrike(9478, "Ripsaw", 1, 16405, core.SpellSchoolPhysical, core.DefenseTypeMelee, 75, 75)
	arcClassicStrike(9511, "Bloodletter Scalpel", 1, 13486, core.SpellSchoolPhysical, core.DefenseTypeMelee, 60, 70)
	arcClassicStrike(10623, "Winter's Bite", 2.2, 13439, core.SpellSchoolFrost, core.DefenseTypeMagic, 20, 30)
	arcClassicStrike(10628, "Deathblow", 1, 16411, core.SpellSchoolPhysical, core.DefenseTypeMelee, 160, 160)
	arcClassicStrike(10761, "Coldrage Dagger", 2.2, 13439, core.SpellSchoolFrost, core.DefenseTypeMagic, 20, 30)
	arcClassicStrike(11603, "Vilerend Slicer", 1, 16405, core.SpellSchoolPhysical, core.DefenseTypeMelee, 75, 75)
	arcClassicStrike(11744, "Bloodfist", 4, 16433, core.SpellSchoolPhysical, core.DefenseTypeMelee, 20, 20)
	arcClassicStrike(12583, "Blackhand Doomsaw", 0.4, 16549, core.SpellSchoolPhysical, core.DefenseTypeMelee, 324, 540)
	arcClassicStrike(13148, "Chillpike", 1, 19260, core.SpellSchoolFrost, core.DefenseTypeMagic, 160, 250)
	arcClassicStrike(13984, "Darrowspike", 1, 18276, core.SpellSchoolFrost, core.DefenseTypeMagic, 90, 90)
	arcClassicStrike(14555, "Alcor's Sunrazor", 1, 18833, core.SpellSchoolFire, core.DefenseTypeMagic, 75, 105)
	// The row adds a point a level above the spell's own level, which the client tables at hand do not
	// state, so the strike keeps its base.
	arcClassicStrike(17054, "Joonho's Mercy", 1, 20883, core.SpellSchoolArcane, core.DefenseTypeMagic, 70, 70)
	arcClassicStrike(17068, "Deathbringer", 1, 18138, core.SpellSchoolShadow, core.DefenseTypeMagic, 110, 140)
	arcClassicStrike(17075, "Vis'kag the Bloodletter", 0.6, 21140, core.SpellSchoolPhysical, core.DefenseTypeMelee, 240, 240)
	arcClassicStrike(17752, "Satyr's Lash", 1, 18205, core.SpellSchoolShadow, core.DefenseTypeMagic, 55, 85)
	// The Fire damage it adds on the target is a flat bonus the sim takes at a spell's power coefficient,
	// and no hunter fire damage carries one.
	arcClassicStrike(12531, "Searing Needle", 1, 16454, core.SpellSchoolFire, core.DefenseTypeMagic, 60, 60)

	// The heal half of the drain changes no damage.
	itemhelpers.CreateWeaponCoHProcDamage(itemhelpers.WeaponProcDamage{
		ItemID:           11920,
		Name:             "Wraith Scythe",
		PPM:              1,
		SpellID:          16414,
		School:           core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		MinDmg:           45,
		MaxDmg:           45,
		BonusCoefficient: 0.3,
	})

	// Ranged equip procs on ranged auto shots and shots, at the rate Forever states for Heartseeking
	// Crossbow's proc of the same kind.
	for _, ranged := range []struct {
		itemID  int32
		name    string
		spellID int32
		school  core.SpellSchool
		minDmg  float64
		maxDmg  float64
	}{
		{2099, "Dwarven Hand Cannon", 29639, core.SpellSchoolFire, 33, 49},
		{2824, "Hurricane", 29502, core.SpellSchoolFrost, 31, 45},
		{10567, "Quillshooter", 29646, core.SpellSchoolNature, 66, 98},
	} {
		itemhelpers.CreateWeaponEquipProcDamage(itemhelpers.WeaponProcDamage{
			ItemID:      ranged.itemID,
			Name:        ranged.name,
			PPM:         3,
			SpellID:     ranged.spellID,
			School:      ranged.school,
			DefenseType: core.DefenseTypeRanged,
			MinDmg:      ranged.minDmg,
			MaxDmg:      ranged.maxDmg,
		})
	}

	// Heart of Wyrmthalak: an equip proc on melee hits.
	shared.NewProcDamageEffect(shared.ProcDamageEffect{
		ItemID:      22321,
		SpellID:     27655,
		School:      core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		MinDmg:      120,
		MaxDmg:      180,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagProc,
		Trigger: core.ProcTrigger{
			Name:     "Heart of Wyrmthalak Proc",
			Callback: core.CallbackOnSpellHitDealt,
			ProcMask: core.ProcMaskMelee,
			Outcome:  core.OutcomeLanded,
		},
		TriggerDPM: func(character *core.Character) *core.DynamicProcManager {
			return character.NewLegacyPPMManager(0.4, core.ProcMaskMelee)
		},
	})

	// Blade of Eternal Darkness: 10% on a landed damaging spell. The mana half changes no damage.
	shared.NewProcDamageEffect(shared.ProcDamageEffect{
		ItemID:  17780,
		SpellID: 27860,
		School:  core.SpellSchoolShadow,
		MinDmg:  100,
		MaxDmg:  100,
		Flags:   core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagProc,
		Trigger: core.ProcTrigger{
			Name:       "Engulfing Shadows Proc",
			Callback:   core.CallbackOnSpellHitDealt,
			ProcMask:   core.ProcMaskSpellDamage,
			Outcome:    core.OutcomeLanded,
			ProcChance: 0.1,
		},
	})
}

// A chance-on-hit proc that opens with an optional direct roll and lays a damage over time on the target.
type arcClassicDot struct {
	itemID  int32
	name    string
	ppm     float64
	spellID int32
	label   string
	school  core.SpellSchool
	defense core.DefenseType
	minHit  float64
	maxHit  float64
	tick    float64
	period  time.Duration
	ticks   int32
	coef    float64
	// Armor the target loses while the damage over time is up.
	armor float64
}

func registerArcClassicDot(d arcClassicDot) {
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: d.itemID,
		Name:   d.name,
		PPM:    d.ppm,
		Spell: func(character *core.Character) *core.Spell {
			return character.RegisterSpell(arcClassicDotConfig(d))
		},
	})
}

func arcClassicDotConfig(d arcClassicDot) core.SpellConfig {
	hit, hitAndCrit := arcClassicOutcomes(d.defense)
	dotAura := core.Aura{Label: d.label}
	if d.armor != 0 {
		armor := d.armor
		dotAura.OnGain = func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, -armor)
		}
		dotAura.OnExpire = func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.Armor, armor)
		}
	}
	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: d.spellID},
		SpellSchool:      d.school,
		DefenseType:      d.defense,
		ProcMask:         core.ProcMaskEmpty,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Dot: core.DotConfig{
			Aura:             dotAura,
			NumberOfTicks:    d.ticks,
			TickLength:       d.period,
			BonusCoefficient: d.coef,
			OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, d.tick)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			var result *core.SpellResult
			if d.maxHit > 0 {
				result = spell.CalcAndDealDamage(sim, target, sim.Roll(d.minHit, d.maxHit), hitAndCrit(spell))
			} else {
				result = spell.CalcAndDealOutcome(sim, target, hit(spell))
			}
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
		},
	}
}

// The hit-only and the hit-and-crit tables of a proc's defense type, a melee one rolling the melee special
// table and any other the spell table.
func arcClassicOutcomes(defense core.DefenseType) (hit func(*core.Spell) core.OutcomeApplier, hitAndCrit func(*core.Spell) core.OutcomeApplier) {
	if defense == core.DefenseTypeMelee {
		return func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMeleeSpecialHit },
			func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMeleeSpecialHitAndCrit }
	}
	return func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMagicHit },
		func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMagicHitAndCrit }
}

func registerArcClassicDots() {
	for _, d := range []arcClassicDot{
		{itemID: 809, name: "Bloodrazor", ppm: 1, spellID: 17504, label: "Rend (Bloodrazor)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 12, period: time.Second * 3, ticks: 10},
		{itemID: 2291, name: "Kang the Decapitator", ppm: 1, spellID: 17153, label: "Rend (Kang the Decapitator)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 56, period: time.Second * 3, ticks: 10},
		{itemID: 5616, name: "Gutwrencher", ppm: 1, spellID: 16406, label: "Rend (Gutwrencher)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 8, period: time.Second * 3, ticks: 10},
		// The Strength it takes from the target changes no damage.
		{itemID: 9372, name: "Sul'thraze the Lasher", ppm: 1, spellID: 11658, label: "Sul'thraze", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, minHit: 90, maxHit: 210, tick: 25, period: time.Second * 3, ticks: 5},
		{itemID: 10797, name: "Firebreather", ppm: 1, spellID: 16413, label: "Fireball (Firebreather)", school: core.SpellSchoolFire, defense: core.DefenseTypeMagic, minHit: 70, maxHit: 70, tick: 3, period: time.Second * 2, ticks: 3},
		{itemID: 10803, name: "Blade of the Wretched", ppm: 1, spellID: 18088, label: "Corruption (Blade of the Wretched)", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 30, period: time.Second, ticks: 3},
		{itemID: 11635, name: "Hookfang Shanker", ppm: 1, spellID: 13526, label: "Corrosive Poison", school: core.SpellSchoolNature, defense: core.DefenseTypeMagic, tick: 7, period: time.Second * 3, ticks: 10, armor: 50},
		{itemID: 12243, name: "Smoldering Claw", ppm: 1, spellID: 15662, label: "Fireball (Smoldering Claw)", school: core.SpellSchoolFire, defense: core.DefenseTypeMagic, minHit: 135, maxHit: 135, tick: 5, period: time.Second * 2, ticks: 3},
		{itemID: 12463, name: "Drakefang Butcher", ppm: 1, spellID: 14118, label: "Rend (Drakefang Butcher)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 15, period: time.Second * 3, ticks: 10},
		// The attack speed it takes from the target changes no damage the wearer deals.
		{itemID: 12582, name: "Keris of Zul'Serak", ppm: 1, spellID: 16528, label: "Numbing Pain", school: core.SpellSchoolNature, defense: core.DefenseTypeMagic, tick: 10, period: time.Second * 2, ticks: 5},
		{itemID: 12621, name: "Demonfork", ppm: 1, spellID: 16603, label: "Demonfork", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 10, period: time.Second * 5, ticks: 5, coef: 0.1},
		{itemID: 12791, name: "Barman Shanker", ppm: 1, spellID: 13318, label: "Rend (Barman Shanker)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 10, period: time.Second * 3, ticks: 10},
		{itemID: 13285, name: "The Blackrock Slicer", ppm: 1, spellID: 17407, label: "Wound (The Blackrock Slicer)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, minHit: 50, maxHit: 150, tick: 6, period: time.Second, ticks: 25},
		{itemID: 13348, name: "Demonshear", ppm: 1, spellID: 17483, label: "Shadow Bolt (Demonshear)", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, minHit: 150, maxHit: 150, tick: 40, period: time.Second * 2, ticks: 3},
		{itemID: 13361, name: "Skullforge Reaver", ppm: 1.7, spellID: 17484, label: "Skullforge Brand", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 2, period: time.Second, ticks: 30},
		{itemID: 13399, name: "Gargoyle Shredder Talons", ppm: 1, spellID: 18202, label: "Rend (Gargoyle Shredder Talons)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 11, period: time.Second * 3, ticks: 10},
		{itemID: 13408, name: "Soul Breaker", ppm: 1, spellID: 17506, label: "Soul Breaker", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 12, period: time.Second * 3, ticks: 10},
		{itemID: 13983, name: "Gravestone War Axe", ppm: 0.5, spellID: 18289, label: "Creeping Mold", school: core.SpellSchoolNature, defense: core.DefenseTypeMagic, tick: 55, period: time.Second * 3, ticks: 5},
		{itemID: 14541, name: "Barovian Family Sword", ppm: 0.5, spellID: 18652, label: "Siphon Health", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 30, period: time.Second * 3, ticks: 5},
		{itemID: 14576, name: "Ebon Hilt of Marduk", ppm: 1, spellID: 18656, label: "Corruption (Ebon Hilt of Marduk)", school: core.SpellSchoolShadow, defense: core.DefenseTypeMagic, tick: 70, period: time.Second, ticks: 3},
		{itemID: 15853, name: "Windreaper", ppm: 1, spellID: 20586, label: "Windreaper", school: core.SpellSchoolNature, defense: core.DefenseTypeMagic, tick: 10, period: time.Second * 2, ticks: 10},
		{itemID: 17730, name: "Gatorbite Axe", ppm: 1, spellID: 21949, label: "Rend (Gatorbite Axe)", school: core.SpellSchoolPhysical, defense: core.DefenseTypeMelee, tick: 23, period: time.Second * 3, ticks: 10},
		{itemID: 17738, name: "Claw of Celebras", ppm: 1, spellID: 21952, label: "Poison (Claw of Celebras)", school: core.SpellSchoolNature, defense: core.DefenseTypeMagic, tick: 9, period: time.Second * 2, ticks: 10},
	} {
		registerArcClassicDot(d)
	}

	// Flame Wrath: a ring of fire on every enemy near the wearer, who is in melee when a weapon procs. The
	// fire shield around the wearer only answers attacks on the wearer.
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: 11809,
		Name:   "Flame Wrath",
		PPM:    1,
		Spell: func(character *core.Character) *core.Spell {
			return character.RegisterSpell(core.SpellConfig{
				ActionID:         core.ActionID{SpellID: 16559},
				SpellSchool:      core.SpellSchoolFire,
				DefenseType:      core.DefenseTypeMagic,
				ProcMask:         core.ProcMaskEmpty,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
					for _, target := range sim.Encounter.ActiveTargetUnits {
						spell.CalcAndDealDamage(sim, target, sim.Roll(130, 170), spell.OutcomeMagicHitAndCrit)
					}
				},
			})
		},
	})
}

func registerArcClassicBuffProcs() {
	statsProc := func(itemID int32, name string, ppm float64, spellID int32, bonus stats.Stats, duration time.Duration) {
		itemhelpers.CreateWeaponProcAura(itemhelpers.WeaponProcAura{
			ItemID: itemID,
			Name:   name,
			PPM:    ppm,
			Aura: func(character *core.Character) *core.Aura {
				return character.NewTemporaryStatsAura(name, core.ActionID{SpellID: spellID}, bonus, duration).Aura
			},
		})
	}
	statsProc(647, "Destiny", 1, 17152, stats.Stats{stats.Strength: 200}, time.Second*10)
	// Strength of Stone adds flat physical damage to each hit.
	statsProc(9418, "Stoneslayer", 1, 12731, stats.Stats{stats.PhysicalDamage: 10}, time.Second*8)
	statsProc(11817, "Lord General's Sword", 1, 15602, stats.Stats{stats.AttackPower: 50, stats.RangedAttackPower: 50}, time.Second*30)

	// Argent Avenger raises melee attack power only, and only against Undead.
	itemhelpers.CreateWeaponProcAura(itemhelpers.WeaponProcAura{
		ItemID: 13246,
		Name:   "Argent Avenger",
		PPM:    1,
		Aura: func(character *core.Character) *core.Aura {
			return character.RegisterAura(core.Aura{
				Label:    "Argent Avenger",
				ActionID: core.ActionID{SpellID: 17352},
				Duration: time.Second * 10,
			}).AttachMobTypeBonusStats(stats.Stats{stats.AttackPower: 200}, proto.MobType_MobTypeUndead)
		},
	})

	// Run speed matters to a hunter walking between melee and range. Only the highest of the run speed
	// effects counts.
	itemhelpers.CreateWeaponProcAura(itemhelpers.WeaponProcAura{
		ItemID: 18410,
		Name:   "Sprinter's Sword",
		PPM:    1,
		Aura: func(character *core.Character) *core.Aura {
			aura := character.RegisterAura(core.Aura{
				Label:    "Sprinter's Sword",
				ActionID: core.ActionID{SpellID: 22863},
				Duration: time.Second * 10,
			})
			aura.NewActiveMovementSpeedEffect(0.3)
			return aura
		},
	})
	core.NewItemEffect(13505, func(agent core.Agent) {
		character := agent.GetCharacter()
		aura := core.MakePermanent(character.RegisterAura(core.Aura{
			Label:    "Unholy Aura",
			ActionID: core.ActionID{SpellID: 17625},
		}))
		aura.NewActiveMovementSpeedEffect(0.08)
		character.ItemSwap.RegisterProc(13505, aura)
	})

	registerArcBlackbladeOfShahram()
}

// Shahram casts one of six spells at random and departs. Three of them change no damage: a slow and a stun
// on enemies, and a heal over time on the party.
func registerArcBlackbladeOfShahram() {
	itemhelpers.CreateWeaponProcTrigger(itemhelpers.WeaponProcTrigger{
		ItemID:             12592,
		Name:               "Summon Shahram",
		PPM:                1,
		TriggerImmediately: true,
		Handler: func(character *core.Character) core.ProcHandler {
			fist := character.RegisterAura(core.Aura{
				Label:    "Fist of Shahram",
				ActionID: core.ActionID{SpellID: 16601},
				Duration: time.Second * 8,
			}).AttachMultiplyMeleeSpeed(1.3)
			will := character.NewTemporaryStatsAura("Will of Shahram", core.ActionID{SpellID: 16598}, stats.Stats{
				stats.Strength:  50,
				stats.Agility:   50,
				stats.Stamina:   50,
				stats.Intellect: 50,
				stats.Spirit:    50,
			}, time.Second*20)
			// Shahram's own spell, so the wearer's damage done does not reach it.
			flames := character.RegisterSpell(core.SpellConfig{
				ActionID:         core.ActionID{SpellID: 16596},
				SpellSchool:      core.SpellSchoolFire,
				DefenseType:      core.DefenseTypeMagic,
				ProcMask:         core.ProcMaskEmpty,
				Flags:            core.SpellFlagIgnoreAttackerModifiers | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagProc,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
					for _, target := range sim.Encounter.ActiveTargetUnits {
						spell.CalcAndDealDamage(sim, target, sim.Roll(90, 160), spell.OutcomeMagicHitAndCrit)
					}
				},
			})
			return func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				switch int(sim.RandomFloat("Shahram") * 6) {
				case 0:
					fist.Activate(sim)
				case 1:
					will.Activate(sim)
				case 2:
					flames.Cast(sim, result.Target)
				}
			}
		},
	})
}

// A debuff a chance-on-hit proc puts on the target, after a roll on the table of its defense type.
func arcClassicDebuffProc(itemID int32, name string, ppm float64, spellID int32, defense core.DefenseType, makeAura func(target *core.Unit) *core.Aura) {
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: itemID,
		Name:   name,
		PPM:    ppm,
		Spell: func(character *core.Character) *core.Spell {
			debuffs := character.NewEnemyAuraArray(makeAura)
			hit, _ := arcClassicOutcomes(defense)
			return character.RegisterSpell(core.SpellConfig{
				ActionID:         core.ActionID{SpellID: spellID},
				SpellSchool:      core.SpellSchoolPhysical,
				DefenseType:      defense,
				ProcMask:         core.ProcMaskEmpty,
				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					if !spell.CalcAndDealOutcome(sim, target, hit(spell)).Landed() {
						return
					}
					debuff := debuffs.Get(target)
					debuff.Activate(sim)
					if debuff.MaxStacks > 0 {
						debuff.AddStack(sim)
					}
				},
			})
		},
	})
}

// A per-stack amount the target carries while the debuff is up.
func arcClassicStackingDebuff(target *core.Unit, label string, spellID int32, duration time.Duration, maxStacks int32, apply func(sim *core.Simulation, unit *core.Unit, stacks int32)) *core.Aura {
	return target.GetOrRegisterAura(core.Aura{
		Label:     label,
		ActionID:  core.ActionID{SpellID: spellID},
		Duration:  duration,
		MaxStacks: maxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks, newStacks int32) {
			apply(sim, aura.Unit, newStacks-oldStacks)
		},
	})
}

func registerArcClassicDebuffProcs() {
	// The same Puncture Armor from either weapon is one debuff, three stacks deep.
	arcClassicDebuffProc(13286, "Rivenspike", 2, 17315, core.DefenseTypeMelee, func(target *core.Unit) *core.Aura {
		return arcClassicStackingDebuff(target, "Puncture Armor", 17315, time.Second*30, 3, func(sim *core.Simulation, unit *core.Unit, stacks int32) {
			unit.AddStatDynamic(sim, stats.Armor, -200*float64(stacks))
		})
	})
	// A Puncture Armor of its own that does not stack, at Rivenspike's rate.
	arcClassicDebuffProc(9465, "Digmaster 5000", 2, 11791, core.DefenseTypeMelee, func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Puncture Armor (Digmaster 5000)",
			ActionID: core.ActionID{SpellID: 11791},
			Duration: time.Second * 30,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.AddStatDynamic(sim, stats.Armor, -100)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				aura.Unit.AddStatDynamic(sim, stats.Armor, 100)
			},
		})
	})
	// Lacerations adds flat damage to every physical hit the target takes, five stacks deep.
	arcClassicDebuffProc(12469, "Mutilator", 1, 10370, core.DefenseTypeMelee, func(target *core.Unit) *core.Aura {
		return arcClassicStackingDebuff(target, "Lacerations", 10370, time.Minute, 5, func(_ *core.Simulation, unit *core.Unit, stacks int32) {
			unit.PseudoStats.BonusPhysicalDamageTaken += 5 * float64(stacks)
		})
	})
}

// Attack power or spell damage an equipped item gives against some creature types. Two items naming the
// same spell both count.
func arcClassicVersus(itemID int32, label string, spellID int32, bonus stats.Stats, mobTypes ...proto.MobType) {
	core.NewItemEffect(itemID, func(agent core.Agent) {
		character := agent.GetCharacter()
		aura := core.MakePermanent(character.RegisterAura(core.Aura{
			Label:    label,
			ActionID: core.ActionID{SpellID: spellID},
		})).AttachMobTypeBonusStats(bonus, mobTypes...)
		character.ItemSwap.RegisterProc(itemID, aura)
	})
}

func registerArcClassicVersusItems() {
	// Each states melee and ranged attack power alike.
	attackPower := func(amount float64) stats.Stats {
		return stats.Stats{stats.AttackPower: amount, stats.RangedAttackPower: amount}
	}
	undead := proto.MobType_MobTypeUndead
	beast := proto.MobType_MobTypeBeast
	arcClassicVersus(10805, "Eater of the Dead", 18074, attackPower(30), undead)
	arcClassicVersus(11628, "Houndmaster's Bow", 18201, attackPower(24), beast)
	arcClassicVersus(11629, "Houndmaster's Rifle", 18201, attackPower(24), beast)
	arcClassicVersus(12709, "Pip's Skinner", 18067, attackPower(45), beast)
	arcClassicVersus(13209, "Seal of the Dawn", 23930, attackPower(81), undead)
	arcClassicVersus(13212, "Halycon's Spiked Collar", 17482, attackPower(48), beast)
	arcClassicVersus(18310, "Fiendish Machete", 22836, attackPower(36), proto.MobType_MobTypeElemental)
	arcClassicVersus(18758, "Specter's Blade", 18098, attackPower(45), undead)
	arcClassicVersus(23088, "Chestguard of Undead Slaying", 17319, attackPower(81), undead)
	arcClassicVersus(23089, "Tunic of Undead Slaying", 17319, attackPower(81), undead)
	arcClassicVersus(23092, "Wristguards of Undead Slaying", 18098, attackPower(45), undead)
	arcClassicVersus(23093, "Wristwraps of Undead Slaying", 18098, attackPower(45), undead)

	// Spell damage reaches a hunter only through the proc drains that scale with it.
	arcClassicVersus(18346, "Threadbare Trousers", 22849, stats.Stats{stats.SpellDamage: 35}, undead)
	arcClassicVersus(19812, "Rune of the Dawn", 24198, stats.Stats{stats.SpellDamage: 48}, undead)
	arcClassicVersus(23085, "Robe of Undead Cleansing", 24197, stats.Stats{stats.SpellDamage: 48}, undead)
	arcClassicVersus(23091, "Bracers of Undead Cleansing", 28876, stats.Stats{stats.SpellDamage: 26}, undead)
}

// The cooldown, shared category and category cooldown of a classic trinket's use, which the item's
// database entry does not carry.
func arcClassicOnUse(cooldown time.Duration, categoryCooldown time.Duration) *proto.ItemEffect {
	return &proto.ItemEffect{Effect: &proto.ItemEffect_OnUse{OnUse: &proto.OnUseEffect{
		CooldownMs:         int32(cooldown.Milliseconds()),
		CategoryId:         1141,
		CategoryCooldownMs: int32(categoryCooldown.Milliseconds()),
	}}}
}

// A damage trinket on its use's cooldowns, filed as a DPS cooldown.
func arcClassicDamageTrinket(itemID int32, cooldown, categoryCooldown time.Duration, config func(character *core.Character) core.SpellConfig) {
	core.NewItemEffect(itemID, func(agent core.Agent) {
		character := agent.GetCharacter()
		spellConfig := config(character)
		spellConfig.ActionID = core.ActionID{ItemID: itemID}
		spellConfig.Flags |= core.SpellFlagNoOnCastComplete
		castTime := spellConfig.Cast.DefaultCast.CastTime
		spellConfig.Cast = shared.OnUseCast(character, arcClassicOnUse(cooldown, categoryCooldown))
		spellConfig.Cast.DefaultCast.CastTime = castTime
		character.AddMajorCooldown(core.MajorCooldown{
			Spell: character.RegisterSpell(spellConfig),
			Type:  core.CooldownTypeDPS,
		})
	})
}

// A stat trinket on its use's cooldowns.
func arcClassicStatTrinket(itemID int32, label string, bonus stats.Stats, duration, cooldown, categoryCooldown time.Duration) {
	core.NewItemEffect(itemID, func(agent core.Agent) {
		character := agent.GetCharacter()
		core.RegisterTemporaryStatsOnUseCD(character, label, bonus, duration, core.SpellConfig{
			ActionID: core.ActionID{ItemID: itemID},
			Cast:     shared.OnUseCast(character, arcClassicOnUse(cooldown, categoryCooldown)),
		})
	})
}

func registerArcClassicTrinkets() {
	arcClassicStatTrinket(19991, "Devilsaur Fury", stats.Stats{
		stats.AttackPower:        150,
		stats.RangedAttackPower:  150,
		stats.PhysicalHitPercent: 2,
	}, time.Second*20, time.Minute*2, time.Second*20)
	// The healing half changes no damage.
	arcClassicStatTrinket(22268, "Chromatic Infusion", stats.Stats{
		stats.SpellDamage:  100,
		stats.HealingPower: 190,
	}, time.Second*15, time.Second*75, time.Second*15)

	arcClassicDamageTrinket(13213, time.Minute*2, time.Second*10, func(character *core.Character) core.SpellConfig {
		return core.SpellConfig{
			SpellSchool:      core.SpellSchoolNature,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagPoison,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			Dot: core.DotConfig{
				Aura:          core.Aura{Label: "Poison (Smolderweb's Eye)", ActionID: core.ActionID{SpellID: 17330}},
				NumberOfTicks: 10,
				TickLength:    time.Second * 2,
				OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.Snapshot(target, 20)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				if spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit).Landed() {
					spell.Dot(target).Apply(sim)
				}
			},
		}
	})

	// A cone 10 yd in front of the wearer.
	arcClassicDamageTrinket(13171, time.Minute*5, time.Second*10, func(character *core.Character) core.SpellConfig {
		return core.SpellConfig{
			SpellSchool:      core.SpellSchoolFire,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			MaxRange:         10 + core.ForeverAreaReachBonus(),
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				for _, target := range sim.Encounter.ActiveTargetUnits {
					spell.CalcAndDealDamage(sim, target, 125, spell.OutcomeMagicHitAndCrit)
				}
			},
		}
	})

	// The bolts strike everything within 10 yd one second after the use, on the use's single tick.
	arcClassicDamageTrinket(13515, time.Minute*5, time.Second*10, func(character *core.Character) core.SpellConfig {
		bolts := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 17669},
			SpellSchool:      core.SpellSchoolNature,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				for _, target := range sim.Encounter.ActiveTargetUnits {
					spell.CalcAndDealDamage(sim, target, 200, spell.OutcomeMagicHitAndCrit)
				}
			},
		})
		return core.SpellConfig{
			SpellSchool: core.SpellSchoolNature,
			ProcMask:    core.ProcMaskEmpty,
			MaxRange:    10 + core.ForeverAreaReachBonus(),
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
				sim.AddPendingAction(core.NewDelayedAction(core.DelayedActionOptions{
					DoAt: sim.CurrentTime + time.Second,
					OnAction: func(sim *core.Simulation) {
						bolts.Cast(sim, target)
					},
				}))
			},
		}
	})

	// A one-second cast that poisons every enemy within 8 yd.
	arcClassicDamageTrinket(10455, time.Minute*15, time.Second*45, func(character *core.Character) core.SpellConfig {
		return core.SpellConfig{
			SpellSchool:      core.SpellSchoolNature,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagPoison,
			MaxRange:         8 + core.ForeverAreaReachBonus(),
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			Cast:             core.CastConfig{DefaultCast: core.Cast{CastTime: time.Second}},
			Dot: core.DotConfig{
				Aura:          core.Aura{Label: "Poison Cloud", ActionID: core.ActionID{SpellID: 12766}},
				NumberOfTicks: 9,
				TickLength:    time.Second * 5,
				OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.Snapshot(target, 50)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				for _, target := range sim.Encounter.ActiveTargetUnits {
					if spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit).Landed() {
						spell.Dot(target).Apply(sim)
					}
				}
			},
		}
	})
}

// An aura on each of the wearer's tamed pets while the item is worn. Checked on reset rather than made
// permanent: a pet resets after its owner's item swap has settled.
func arcClassicPetItem(itemID int32, label string, spellID int32, attach func(pet *core.Pet, aura *core.Aura)) {
	core.NewItemEffect(itemID, func(agent core.Agent) {
		character := agent.GetCharacter()
		slots := character.ItemSwap.EligibleSlotsForItem(itemID)
		for _, pet := range character.Pets {
			if pet.IsGuardian() {
				continue
			}
			aura := pet.RegisterAura(core.Aura{
				Label:    label,
				ActionID: core.ActionID{SpellID: spellID},
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					if character.HasItemEquipped(itemID, slots) {
						aura.Activate(sim)
					}
				},
			})
			attach(pet, aura)
			character.ItemSwap.RegisterProcWithSlots(itemID, aura, slots)
		}
	})
}

func registerArcClassicPetItems() {
	arcClassicPetItem(22015, "Increased Pet Crit +2%", 27042, func(_ *core.Pet, aura *core.Aura) {
		aura.AttachStatBuff(stats.PhysicalCritPercent, 2)
	})
	arcClassicPetItem(22061, "Increased Pet Damage +3%", 27205, func(pet *core.Pet, aura *core.Aura) {
		aura.AttachMultiplicativePseudoStatBuff(&pet.PseudoStats.DamageDealtMultiplier, 1.03)
	})

	// Devilsaur Tooth: the pet's next melee or ranged attack crits. The use takes the global cooldown.
	core.NewItemEffect(19992, func(agent core.Agent) {
		character := agent.GetCharacter()
		var instincts []*core.Aura
		for _, pet := range character.Pets {
			if pet.IsGuardian() {
				continue
			}
			instincts = append(instincts, pet.RegisterAura(core.Aura{
				Label:    "Primal Instinct",
				ActionID: core.ActionID{SpellID: 24353},
				Duration: core.NeverExpires,
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if spell.ProcMask.Matches(core.ProcMaskMeleeOrRanged) && result.Landed() {
						aura.Deactivate(sim)
					}
				},
			}).AttachStatBuff(stats.PhysicalCritPercent, 100))
		}
		if len(instincts) == 0 {
			return
		}
		cast := shared.OnUseCast(character, arcClassicOnUse(time.Minute*2, time.Second*10))
		cast.DefaultCast.GCD = core.GCDDefault
		character.AddMajorCooldown(core.MajorCooldown{
			Type: core.CooldownTypeDPS,
			Spell: character.RegisterSpell(core.SpellConfig{
				ActionID: core.ActionID{ItemID: 19992},
				Flags:    core.SpellFlagNoOnCastComplete,
				Cast:     cast,
				ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
					for _, instinct := range instincts {
						if instinct.Unit.IsEnabled() {
							return true
						}
					}
					return false
				},
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
					for _, instinct := range instincts {
						if instinct.Unit.IsEnabled() {
							instinct.Activate(sim)
						}
					}
				},
			}),
		})
	})
}
