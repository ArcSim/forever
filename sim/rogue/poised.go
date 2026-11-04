package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var poisedRank = spelldata.MustFind(1328826)

// Out of combat at full Energy the rogue is Poised, so the opener is the one ability it cheapens:
// the fight starts at full Energy and nothing in it leaves combat. Composure deepens the discount.
func (rogue *Rogue) registerPoised() {
	discount := -poisedRank.EffectN(1).BaseValue() - spellData.Composure.EffectAt(1).ValueAt(rogue.Talents.Composure)

	rogue.RegisterAura(core.Aura{
		Label:    "Poised",
		ActionID: core.ActionID{SpellID: poisedRank.ID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.Matches(RogueSpellsAll) && spell.Cost != nil && spell.Cost.BaseCost > 0 {
				aura.Deactivate(sim)
			}
		},
	}).AttachSpellMod(core.SpellModConfig{
		Kind:      core.SpellMod_PowerCost_Flat,
		ClassMask: RogueSpellsAll,
		IntValue:  -int32(discount),
	})
}
