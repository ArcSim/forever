package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var shankRank = spellData.Shank.Highest()

const shankEnergyTag = 1

// An off-hand strike that pays Energy rather than a Combo Point. Flash of Steel adds Energy and takes
// time off the cooldown, Boot Knife gives it a chance at Combo Points and Venom hangs its poison buff on it.
func (rogue *Rogue) registerShankSpell() {
	energy := shankRank.EffectN(2).BaseValue() + spellData.FlashOfSteel.EffectAt(1).ValueAt(rogue.Talents.FlashOfSteel)
	// Its own tag: the cast's cost already books energy under the spell's id.
	energyMetrics := rogue.NewEnergyMetrics(core.ActionID{SpellID: shankRank.ID}.WithTag(shankEnergyTag))

	bootKnifeChance := spellData.BootKnife.EffectAt(1).FractionAt(rogue.Talents.BootKnife)
	bootKnifePoints := int32(spellData.BootKnifeTriggered.Highest().EnergizeEffect().Average())
	var bootKnifeMetrics *core.ResourceMetrics
	if rogue.Talents.BootKnife > 0 {
		bootKnifeMetrics = rogue.NewComboPointMetrics(core.ActionID{SpellID: spellData.BootKnife.Rank(rogue.Talents.BootKnife).ID})
	}

	config := spelldata.SpellConfig(&rogue.Unit, shankRank, spelldata.Melee(core.ProcMaskMeleeOHSpecial))
	config.ClassSpellMask = RogueSpellShank
	// Only a Shank that can add Combo Points counts as an ability that adds them, for Seal Fate.
	if rogue.Talents.BootKnife > 0 {
		config.Flags |= SpellFlagBuilder
	}
	config.DamageMultiplier = shankRank.EffectN(1).Percent()
	config.Cast.CD.Duration += time.Duration(spellData.FlashOfSteel.EffectAt(2).ValueAt(rogue.Talents.FlashOfSteel)) * time.Millisecond
	config.ExtraCastCondition = func(sim *core.Simulation, target *core.Unit) bool {
		return rogue.AutoAttacks.IsDualWielding
	}

	alwaysHits := shankRank.AlwaysHits()
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		outcome := spell.OutcomeMeleeWeaponSpecialHitAndCrit
		if alwaysHits {
			outcome = spell.OutcomeMeleeSpecialCritOnly
		}
		baseDamage := ohStrikeDamage(shankRank, &rogue.Unit, sim, spell.MeleeAttackPower(target))
		result := spell.CalcAndDealDamage(sim, target, baseDamage, outcome)
		if !result.Landed() {
			spell.IssueRefund(sim)
			return
		}
		rogue.AddEnergy(sim, energy, energyMetrics)
		if bootKnifeChance > 0 && sim.Proc(bootKnifeChance, "Boot Knife") {
			rogue.AddComboPoints(sim, bootKnifePoints, bootKnifeMetrics)
		}
		rogue.VenomAura.Activate(sim)
	}

	rogue.Shank = rogue.RegisterSpell(config)
}
