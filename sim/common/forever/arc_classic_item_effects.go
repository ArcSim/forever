package forever

import (
	"time"

	"github.com/wowsims/forever/sim/common/itemhelpers"
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// Effects of classic items the Forever client data does not carry (see tools/database/arc_classic_items.go),
// so no generated registration exists for them.
func RegisterArcClassicItemEffects() {
	// Hand of Justice: a melee hit has a 1% chance of an extra main-hand swing, 2 s cooldown. Forever's
	// tooltip is a third of the row's 3%.
	core.NewItemEffect(11815, func(agent core.Agent) {
		character := agent.GetCharacter()
		if !character.AutoAttacks.AutoSwingMelee {
			return
		}
		trigger := core.ProcTrigger{
			Name:       "Hand of Justice",
			Callback:   core.CallbackOnSpellHitDealt,
			Outcome:    core.OutcomeLanded,
			ProcMask:   core.ProcMaskMelee,
			ProcChance: 0.01,
			ICD:        time.Second * 2,
		}
		onInit := character.ExtraAttackProc(&trigger, 15601, 1)
		character.MakeProcTriggerAura(trigger).ApplyOnInit(onInit)
	})

	// Felstriker: chance on hit, every attack lands and crits for 3 s (1 PPM).
	itemhelpers.CreateWeaponProcAura(itemhelpers.WeaponProcAura{
		ItemID: 12590,
		Name:   "Felstriker",
		PPM:    1,
		Aura: func(character *core.Character) *core.Aura {
			return character.NewTemporaryStatsAura("Felstriker", core.ActionID{SpellID: 16551}, stats.Stats{
				stats.CritRating: 100 * core.CritRatingPerCritPercent,
				stats.HitRating:  100 * core.HitRatingPerHitPercent,
			}, time.Second*3).Aura
		},
	})

	// Bonereaver's Edge: chance on hit, attacks ignore 700 armor for 10 s, stacking to 3 (2 PPM). Each proc
	// adds one stack.
	itemhelpers.CreateWeaponProcSpell(itemhelpers.WeaponProcSpell{
		ItemID: 17076,
		Name:   "Bonereaver's Edge",
		PPM:    2,
		Spell: func(character *core.Character) *core.Spell {
			actionID := core.ActionID{SpellID: 21153}
			buff := character.RegisterAura(core.Aura{
				ActionID:  actionID,
				Label:     "Bonereaver's Edge",
				Duration:  time.Second * 10,
				MaxStacks: 3,
				OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks, newStacks int32) {
					for _, target := range sim.Encounter.ActiveTargetUnits {
						target.AddStatDynamic(sim, stats.Armor, 700*float64(oldStacks-newStacks))
					}
				},
			})
			return character.RegisterSpell(core.SpellConfig{
				ActionID:    actionID,
				SpellSchool: core.SpellSchoolShadow,
				ProcMask:    core.ProcMaskEmpty,
				ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
					buff.Activate(sim)
					buff.AddStack(sim)
				},
			})
		},
	})

	// Grand Crusader (one-hand) and Grand Inquisitor (two-hand), Forever's stronger Crusaders: the client carries no
	// proc rate for them, so they keep Crusader's 1 PPM, one Strength buff a hand.
	registerArcGrandCrusader(7940, "Grand Crusader", 1231124, 120)
	registerArcGrandCrusader(7943, "Grand Inquisitor", 1232169, 200)

	// Blackhand's Breadth, Forever version: +1% crit (in the item override) and an on-use +5% crit for
	// 20 s, 5 min cooldown.
	core.NewItemEffect(13965, func(agent core.Agent) {
		character := agent.GetCharacter()
		effect := &proto.ItemEffect{
			BuffName:         "Blackhand's Breadth",
			EffectDurationMs: 20000,
			Effect:           &proto.ItemEffect_OnUse{OnUse: &proto.OnUseEffect{CooldownMs: 300000}},
		}
		core.RegisterTemporaryStatsOnUseCD(character, effect.BuffName, stats.Stats{stats.CritRating: 5 * core.CritRatingPerCritPercent},
			time.Second*20, core.SpellConfig{
				ActionID: core.ActionID{ItemID: 13965},
				Cast:     shared.OnUseCast(character, effect),
			})
	})
}

func registerArcGrandCrusader(effectID int32, label string, spellID int32, strength float64) {
	core.NewEnchantEffect(effectID, func(agent core.Agent) {
		character := agent.GetCharacter()
		actionID := core.ActionID{SpellID: spellID}
		handAura := func(slot proto.ItemSlot) *core.StatBuffAura {
			aura := character.NewTemporaryStatsAura(label+core.Ternary(slot == proto.ItemSlot_ItemSlotMainHand, " (MH)", " (OH)"), actionID.WithHand(slot),
				stats.Stats{stats.Strength: strength}, time.Second*20)
			character.AddStatProcBuff(effectID, aura, true, []proto.ItemSlot{slot})
			character.ItemSwap.RegisterWeaponEnchantBuff(aura.Aura, effectID)
			return aura
		}
		mhAura := handAura(proto.ItemSlot_ItemSlotMainHand)
		ohAura := handAura(proto.ItemSlot_ItemSlotOffHand)
		character.MakeProcTriggerAura(core.ProcTrigger{
			Name:         "Enchant Weapon - " + label,
			ActionID:     actionID,
			IsWeaponProc: true,
			DPM:          character.NewDynamicLegacyProcForEnchant(effectID, 1, 0),
			Outcome:      core.OutcomeLanded,
			Callback:     core.CallbackOnSpellHitDealt,
			Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
				core.Ternary(spell.IsOH(), ohAura, mhAura).Activate(sim)
			},
		})
	})
}

// Leatherworking - Leather
// https://www.wowhead.com/forever/item-set=143/devilsaur-armor
//
// The 2-piece bonus is +2% hit and +2% spell hit in the client data.
var ItemSetDevilsaurArmor = core.NewItemSet(core.ItemSet{
	ID:   143,
	Name: "Devilsaur Armor",
	Bonuses: map[int32]core.ApplySetBonus{
		2: func(_ core.Agent, setBonusAura *core.Aura) {
			setBonusAura.AttachStatsBuff(stats.Stats{stats.PhysicalHitPercent: 2, stats.SpellHitPercent: 2})
		},
	},
})
