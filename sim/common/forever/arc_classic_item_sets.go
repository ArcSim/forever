package forever

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// Sets of classic dungeon pieces the Forever client does not carry as items. Forever's own set table still
// lists them, with bonuses of its own where it reworked the set, and those are the bonuses here. A bonus that
// only answers attacks on the wearer, lowers a crowd-control duration or shortens a cooldown the hunter
// never uses has no place in the sim.

// The Savage Gladiator pieces.
var ItemSetTheGladiator = core.NewItemSet(core.ItemSet{
	ID:    1,
	Name:  "The Gladiator",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		3: shared.ParsedSetBonus(spelldata.MustFind(9331)), // +20 attack power
		4: arcClassicMeleeSetProc(1301123, "Roar of the Crowd", func(character *core.Character) core.ProcHandler {
			row := spelldata.MustFind(1301124)
			buff := character.RegisterAura(spelldata.AuraConfig(row))
			spelldata.ParseEffects(character, buff, row)
			return func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
				buff.Activate(sim)
			}
		}),
		5: shared.ParsedSetBonus(spelldata.MustFind(1314795)), // +1% crit
	},
})

var ItemSetDalRendsArms = core.NewItemSet(core.ItemSet{
	ID:    41,
	Name:  "Dal'Rend's Arms",
	Slots: []proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand, proto.ItemSlot_ItemSlotOffHand},
	Bonuses: map[int32]core.ApplySetBonus{
		2: shared.ParsedSetBonus(spelldata.MustFind(14056)), // +50 attack power
	},
})

var ItemSetThePostmaster = core.NewItemSet(core.ItemSet{
	ID:   81,
	Name: "The Postmaster",
	Slots: []proto.ItemSlot{
		proto.ItemSlot_ItemSlotHead,
		proto.ItemSlot_ItemSlotChest,
		proto.ItemSlot_ItemSlotLegs,
		proto.ItemSlot_ItemSlotFeet,
		proto.ItemSlot_ItemSlotFinger1,
		proto.ItemSlot_ItemSlotFinger2,
	},
	Bonuses: map[int32]core.ApplySetBonus{
		// Run speed, which bids with the other run speed effects for the highest.
		2: func(_ core.Agent, setBonusAura *core.Aura) {
			setBonusAura.NewActiveMovementSpeedEffect(spelldata.MustFind(1302376).EffectN(1).Percent())
			setBonusAura.ExposeToAPL(1302376)
		},
		3: shared.ParsedSetBonus(spelldata.MustFind(14047)),  // +23 spell damage
		5: shared.ParsedSetBonus(spelldata.MustFind(432639)), // +1% hit
	},
})

var ItemSetCadaverousGarb = core.NewItemSet(core.ItemSet{
	ID:    121,
	Name:  "Cadaverous Garb",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		3: shared.ParsedSetBonus(spelldata.MustFind(9140)),   // +10 attack power
		4: shared.ParsedSetBonus(spelldata.MustFind(18676)),  // +5 all resistances
		5: shared.ParsedSetBonus(spelldata.MustFind(460230)), // +2% hit
	},
})

// The 3-piece drain procs off spells at a rate the client does not state, and a spell has no weapon speed to
// turn a rate per minute into a chance.
var ItemSetNecropileRaiment = core.NewItemSet(core.ItemSet{
	ID:    122,
	Name:  "Necropile Raiment",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		2: shared.ParsedSetBonus(spelldata.MustFind(1299734)), // +0.5% hit
		4: shared.ParsedSetBonus(spelldata.MustFind(18676)),   // +5 all resistances
		5: shared.ParsedSetBonus(spelldata.MustFind(14047)),   // +23 spell damage
	},
})

var ItemSetBloodmailRegalia = core.NewItemSet(core.ItemSet{
	ID:    123,
	Name:  "Bloodmail Regalia",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		2: shared.ParsedSetBonus(spelldata.MustFind(9140)), // +10 attack power
		// A bleed on the target of a melee swing.
		3: arcClassicMeleeSetProc(1299740, "Bloodmail", func(character *core.Character) core.ProcHandler {
			row := spelldata.MustFind(1299739)
			config := spelldata.SpellConfig(&character.Unit, row, spelldata.Melee(core.ProcMaskEmpty), spelldata.Proc())
			config.Flags |= core.SpellFlagProc
			config.Dot = spelldata.DotConfig(row, row.PeriodicDamageEffect(), spelldata.Label("Bloodmail"))
			config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				if spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit).Landed() {
					spell.Dot(target).Apply(sim)
				}
			}
			bleed := character.RegisterSpell(config)
			return func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				bleed.Cast(sim, result.Target)
			}
		}),
		4: shared.ParsedSetBonus(spelldata.MustFind(18676)),   // +5 all resistances
		5: shared.ParsedSetBonus(spelldata.MustFind(1299738)), // +1.5% crit
	},
})

// The Premier Chain pieces carry the set's id in Forever.
var ItemSetChampionsPursuit = core.NewItemSet(core.ItemSet{
	ID:    361,
	Name:  "Champion's Pursuit",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		2: shared.ParsedSetBonus(spelldata.MustFind(14384)), // +20 Agility
		6: shared.ParsedSetBonus(spelldata.MustFind(14467)), // +20 Stamina
	},
})

// Three-piece Naxxramas sets: 2% more damage against Undead.
var (
	ItemSetRegaliaOfUndeadCleansing = arcClassicUndeadSet(536, "Regalia of Undead Cleansing")
	ItemSetGarbOfTheUndeadSlayer    = arcClassicUndeadSet(535, "Garb of the Undead Slayer")
	ItemSetUndeadSlayersArmor       = arcClassicUndeadSet(534, "Undead Slayer's Armor")
)

func arcClassicUndeadSet(id int32, name string) *core.ItemSet {
	return core.NewItemSet(core.ItemSet{
		ID:    id,
		Name:  name,
		Slots: shared.EightPieceSlots,
		Bonuses: map[int32]core.ApplySetBonus{
			3: shared.ParsedSetBonus(spelldata.MustFind(29068)),
		},
	})
}

// Dungeon Set 2's hunter set, which shares its name with the reissue of id 1669. The 4-piece bonus restores
// mana.
var ItemSetBeastmasterArmor = core.NewItemSet(core.ItemSet{
	ID:    515,
	Name:  "Beastmaster Armor",
	Slots: shared.EightPieceSlots,
	Bonuses: map[int32]core.ApplySetBonus{
		2: shared.ParsedSetBonus(dungeonResistances),
		6: shared.ParsedSetBonus(dungeonTwoAttack),
		8: shared.ParsedSetBonus(spelldata.MustFind(14803)), // +200 armor
	},
})

// A set proc on the melee hits its row hears. The client states its rate as procs per minute without the
// number, so it takes the 1 PPM of a classic chance-on-hit proc.
func arcClassicMeleeSetProc(procID int32, name string, handler func(character *core.Character) core.ProcHandler) core.ApplySetBonus {
	return func(agent core.Agent, setBonusAura *core.Aura) {
		character := agent.GetCharacter()
		trigger := spelldata.ProcTrigger(character, spelldata.MustFind(procID), handler(character), spelldata.PPM(1))
		trigger.Name = name + " Proc"
		trigger.TriggerImmediately = true
		setBonusAura.AttachProcTrigger(trigger)
		setBonusAura.ExposeToAPL(procID)
	}
}
