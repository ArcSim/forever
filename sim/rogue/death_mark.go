package rogue

import (
	"math"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

var deathMarkRank = spellData.DeathMark.Highest()

// The tooltip's numbers: no effect row carries them. A Combo Point stores 12% of the damage the rogue
// deals the target ($ratio), Garrote's and Rupture's twice that, and the cap is
// (flat + per point * points + 0.04 * attack power * points) * (1 + melee crit chance) ($maxAP, $mcrit).
const (
	deathMarkStoredPerPoint = 0.12
	deathMarkBleedRate      = 2.0
	deathMarkCapAPPerPoint  = 0.04
)

// Past the cast's combo point splits (tags 0-5).
const deathMarkPayoutTag = 10

// The cap's flat amount and its amount per Combo Point, by rank.
var deathMarkCaps = map[int32][2]float64{
	1316709: {63, 102},
	1325756: {98, 158},
	1325757: {138, 217},
	1325758: {156, 245},
}

// A finisher that marks the target: for the mark's duration a share of the rogue's damage to it is
// stored, and when the mark runs out the stored damage is dealt, up to the cap set at the cast. The
// payout is the damage already dealt, so it takes no armor, no crit, no bonus damage and no modifiers but
// Coup de Grace's.
func (rogue *Rogue) registerDeathMark() {
	if !rogue.Talents.DeathMark {
		return
	}

	capFlat, capPerPoint := deathMarkCaps[deathMarkRank.ID][0], deathMarkCaps[deathMarkRank.ID][1]
	var stored, storeRate, capAmount float64
	var marked *core.Unit

	coupDeGrace := spellData.CoupDeGrace.EffectAt(2).MultiplierAt(rogue.Talents.CoupDeGrace)
	payout := rogue.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: deathMarkRank.ID}.WithTag(deathMarkPayoutTag),
		SpellSchool:      core.SpellSchoolPhysical,
		ProcMask:         core.ProcMaskEmpty,
		Flags:            core.SpellFlagIgnoreResists | core.SpellFlagIgnoreModifiers | core.SpellFlagPassiveSpell,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.NewResult(target)
			result.Damage = math.Min(stored, capAmount) * coupDeGrace
			result.Outcome = core.OutcomeHit
			spell.DealDamage(sim, result)
		},
	})

	store := func(spell *core.Spell, result *core.SpellResult) {
		if spell == payout || result.Target != marked || result.Damage <= 0 {
			return
		}
		rate := storeRate
		if spell.Matches(RogueSpellGarrote | RogueSpellRupture) {
			rate *= deathMarkBleedRate
		}
		stored += result.Damage * rate
	}

	markAura := rogue.RegisterAura(core.Aura{
		Label:    "Death Mark",
		ActionID: core.ActionID{SpellID: deathMarkRank.ID},
		Duration: deathMarkRank.Duration(),
		OnSpellHitDealt: func(_ *core.Aura, _ *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			store(spell, result)
		},
		OnPeriodicDamageDealt: func(_ *core.Aura, _ *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			store(spell, result)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			if stored > 0 {
				payout.Cast(sim, marked)
			}
		},
	})

	config := spelldata.SpellConfig(&rogue.Unit, deathMarkRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))
	config.ExtraCastCondition = rogue.FinisherExtraCastCondition
	config.Flags |= SpellFlagFinisher
	config.ClassSpellMask = RogueSpellDeathMark
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		rogue.BreakStealth(sim)
		result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
		if !result.Landed() {
			spell.IssueRefund(sim)
			return
		}
		points := float64(rogue.ComboPoints())
		stored = 0
		marked = target
		storeRate = deathMarkStoredPerPoint * points
		capAmount = (capFlat + capPerPoint*points + deathMarkCapAPPerPoint*points*spell.MeleeAttackPower(target)) *
			(1 + rogue.GetStat(stats.PhysicalCritPercent)/100)
		markAura.Activate(sim)
		rogue.ApplyFinisher(sim, spell)
	}

	config.MetricSplits = 6
	config.Cast.ModifyCast = func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
		spell.SetMetricsSplit(rogue.ComboPoints())
	}

	rogue.DeathMark = rogue.RegisterSpell(config)
}
