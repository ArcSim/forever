package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (rogue *Rogue) registerSubtletyTalents() {
	// Tier 1
	// Camouflage NYI
	// Master of Deception NYI
	rogue.registerOpportunity()

	// Tier 2
	// Setup NYI
	rogue.registerElusiveness()
	// Dirty Tricks NYI
	rogue.registerImprovedAmbush()

	// Tier 3
	rogue.registerInitiative()
	rogue.registerGhostlyStrike()
	// Improved Distract NYI

	// Tier 4
	// Heightened Senses NYI
	rogue.registerPremeditation()
	rogue.registerSerratedBlades()

	// Tier 5
	rogue.registerDirtyDeeds()
	rogue.registerPreparation()
	rogue.registerHemorrhage()

	// Tier 6
	// Boot Knife modeled in Shank

	// Tier 7
	rogue.registerDeathMark()
}

func (rogue *Rogue) registerOpportunity() {
	if rogue.Talents.Opportunity == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.Opportunity.Rank(rogue.Talents.Opportunity))
}

func (rogue *Rogue) registerInitiative() {
	if rogue.Talents.Initiative == 0 {
		return
	}

	initiativeRank := spellData.Initiative.Rank(rogue.Talents.Initiative)
	initMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: initiativeRank.ID})

	trigger := spelldata.ProcTrigger(&rogue.Character, initiativeRank,
		func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			rogue.AddComboPoints(sim, 1, initMetrics)
		})
	trigger.ClassSpellMask = RogueSpellGarrote | RogueSpellAmbush
	trigger.TriggerImmediately = true
	trigger.Outcome = core.OutcomeLanded
	trigger.RequireDamageDealt = false
	rogue.MakeProcTriggerAura(trigger)
}

var ghostlyStrikeRank = spellData.GhostlyStrike.Highest()

func (rogue *Rogue) registerGhostlyStrike() {
	if !rogue.Talents.GhostlyStrike {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, ghostlyStrikeRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.Flags |= SpellFlagBuilder
	config.ClassSpellMask = RogueSpellGhostlyStrike
	config.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), ghostlyStrikeRank.EffectN(4).Percent(), ghostlyStrikeRank.EffectN(1).Percent())

	// Dodge aura not implemented; maybe if rogue tanking comes back

	rogue.registerOpenerUnstealthed()

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		baseDamage := mhStrikeDamage(ghostlyStrikeRank, spell.Unit, sim, spell.MeleeAttackPower(target))

		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

		if result.Landed() {
			rogue.AddComboPoints(sim, int32(ghostlyStrikeRank.ProcEnergizeEffect().Roll(sim)), spell.ComboPointMetrics())
			rogue.OpenerUnstealthedAura.Activate(sim)
		} else {
			spell.IssueRefund(sim)
		}
	}

	rogue.GhostlyStrike = rogue.RegisterSpell(config)

	rogue.RegisterItemSwapCallback([]proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand}, func(s *core.Simulation, slot proto.ItemSlot) {
		rogue.GhostlyStrike.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), ghostlyStrikeRank.EffectN(4).Percent(), ghostlyStrikeRank.EffectN(1).Percent())
	})
}

func (rogue *Rogue) registerImprovedAmbush() {
	if rogue.Talents.ImprovedAmbush == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.ImprovedAmbush.Rank(rogue.Talents.ImprovedAmbush))
}

func (rogue *Rogue) registerElusiveness() {
	if rogue.Talents.Elusiveness == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.Elusiveness.Rank(rogue.Talents.Elusiveness))
}

func (rogue *Rogue) registerSerratedBlades() {
	if rogue.Talents.SerratedBlades == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.SerratedBlades.Rank(rogue.Talents.SerratedBlades))
}

func (rogue *Rogue) registerPreparation() {
	if !rogue.Talents.Preparation {
		return
	}

	// Prep purely runs a server-side script, so no client data to reference.

	config := spelldata.SpellConfig(&rogue.Unit, spellData.Preparation.Highest(), spelldata.Flags(core.SpellFlagAPL|core.SpellFlagMCD))
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		if rogue.ColdBlood != nil {
			rogue.ColdBlood.CD.Set(0)
		}
		if rogue.Premeditation != nil {
			rogue.Premeditation.CD.Set(0)
		}
		if rogue.Vanish != nil {
			rogue.Vanish.CD.Set(0)
		}
		if rogue.GhostlyStrike != nil {
			rogue.GhostlyStrike.CD.Set(0)
		}
		if rogue.BladeFlurry != nil {
			rogue.BladeFlurry.CD.Set(0)
		}
		if rogue.AdrenalineRush != nil {
			rogue.AdrenalineRush.CD.Set(0)
		}
	}

	rogue.Preparation = rogue.RegisterSpell(config)

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.Preparation,
		Type:  core.CooldownTypeDPS,
	})
}

func (rogue *Rogue) registerDirtyDeeds() {
	if rogue.Talents.DirtyDeeds == 0 {
		return
	}

	spelldata.ParseStatic(&rogue.Character,
		spellData.DirtyDeeds.Rank(rogue.Talents.DirtyDeeds))
}

var hemorrhageRank = spellData.Hemorrhage.Highest()

func (rogue *Rogue) registerHemorrhage() {
	if !rogue.Talents.Hemorrhage {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, hemorrhageRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.Flags |= SpellFlagBuilder
	config.ClassSpellMask = RogueSpellHemorrhage
	config.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), hemorrhageRank.EffectN(5).Percent(), hemorrhageRank.EffectN(4).Percent())

	hemoAuras := rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		hemoAura := target.GetOrRegisterAura(spelldata.AuraConfig(hemorrhageRank))
		spelldata.ParseEffects(nil, hemoAura, spellData.Hemorrhage.Highest(),
			spelldata.FromCaster(&rogue.Unit, DDBC_Hemorrhage, DDBC_Total))
		return hemoAura
	})

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		baseDamage := rogue.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
		result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		if result.Landed() {
			hemoAuras.Get(target).Activate(sim)
			rogue.AddComboPoints(sim, int32(hemorrhageRank.ProcEnergizeEffect().Roll(sim)), spell.ComboPointMetrics())
		} else {
			spell.IssueRefund(sim)
		}
	}

	rogue.Hemorrhage = rogue.GetOrRegisterSpell(config)

	rogue.RegisterItemSwapCallback([]proto.ItemSlot{proto.ItemSlot_ItemSlotMainHand}, func(s *core.Simulation, slot proto.ItemSlot) {
		rogue.Hemorrhage.DamageMultiplier = core.TernaryFloat64(rogue.HasDagger(core.MainHand), hemorrhageRank.EffectN(5).Percent(), hemorrhageRank.EffectN(4).Percent())
	})
}

var premeditationRank = spellData.Premeditation.Highest()

func (rogue *Rogue) registerPremeditation() {
	if !rogue.Talents.Premeditation {
		return
	}

	config := spelldata.SpellConfig(&rogue.Unit, premeditationRank, spelldata.Flags(core.SpellFlagAPL|core.SpellFlagMCD))
	premedMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: premeditationRank.ID})

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.AddComboPoints(sim, int32(premeditationRank.ProcEnergizeEffect().Roll(sim)), premedMetrics)
	}

	rogue.Premeditation = rogue.GetOrRegisterSpell(config)

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell: rogue.Premeditation,
		Type:  core.CooldownTypeDPS,
	})
}

// Ghostly Strike's buff: the next Ambush or Garrote within its duration needs no Stealth, and the
// cast that uses it spends it.
func (rogue *Rogue) registerOpenerUnstealthed() {
	allowOpeners := func(allowed bool) {
		rogue.Ambush.CastRequirement.CasterForm = allowed
		rogue.Garrote.CastRequirement.CasterForm = allowed
	}
	rogue.OpenerUnstealthedAura = rogue.RegisterAura(spelldata.AuraConfig(spellData.GhostlyStrikeTriggered.Highest())).
		ApplyOnGain(func(_ *core.Aura, _ *core.Simulation) {
			allowOpeners(true)
		}).
		ApplyOnExpire(func(_ *core.Aura, _ *core.Simulation) {
			allowOpeners(false)
		})

	rogue.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Ghostly Strike - Opener Watcher",
		ActionID:           core.ActionID{SpellID: spellData.GhostlyStrikeTriggered.Highest().ID},
		Callback:           core.CallbackOnCastComplete,
		ClassSpellMask:     RogueSpellAmbush | RogueSpellGarrote,
		TriggerImmediately: true,
		ProcChance:         1,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			rogue.OpenerUnstealthedAura.Deactivate(sim)
		},
	})
}
