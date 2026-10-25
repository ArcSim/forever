package druid

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (druid *Druid) registerFeralChargeSpell() {
	rank := spellData.FeralCharge.Highest()
	root := spellData.FeralChargeTriggered.Highest()
	roots := druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(spelldata.AuraConfig(root))
	})
	config := spelldata.SpellConfig(&druid.Unit, rank)
	config.Flags |= core.SpellFlagAPL | core.SpellFlagMovesToTarget
	config.Cast.IgnoreHaste = true
	config.ExtraCastCondition = func(_ *core.Simulation, _ *core.Unit) bool {
		return druid.Talents.FeralCharge
	}
	chargeMovement := core.MakeDashAura(&druid.Unit, "Feral Charge Movement", core.ActionID{SpellID: rank.ID, Tag: 1}, 5*time.Second, nil)
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
		if druid.InForm(Cat) {
			// Timed movement must not restore the position of an earlier walk.
			druid.StopMoving(sim)
			// TODO: Verify the modeled travel speed and landing delay in game.
			travel := core.DurationFromSeconds(druid.DistanceFromTarget / 80)
			druid.ExtendGCDUntil(sim, max(druid.NextGCDAt(), sim.CurrentTime+travel))
			druid.DistanceFromTarget = 0
			druid.MoveDuration(travel, sim)
			landingDelay := core.DurationFromSeconds(0.150 + sim.RandomFloat("Cat Charge")*0.6)
			druid.AutoAttacks.CancelMeleeSwing(sim)
			sim.AddPendingAction(&core.PendingAction{
				NextActionAt: sim.CurrentTime + travel + landingDelay,
				Priority:     core.ActionPriorityDOT,
				OnAction: func(sim *core.Simulation) {
					druid.AutoAttacks.EnableMeleeSwing(sim)
				},
			})
			return
		}
		if !druid.InForm(Bear) {
			return
		}
		chargeMovement.Activate(sim)
		druid.MoveTo(core.MaxMeleeRange-1, sim) // Overshoot the discrete range updates to reach melee.
		roots.Get(target).Activate(sim)
	}
	config.RelatedAuraArrays = roots.ToMap()
	druid.FeralCharge = druid.RegisterSpell(config)
	if druid.CatFormAura != nil {
		gcd := druid.FeralCharge.DefaultCast.GCD
		druid.CatFormAura.ApplyOnGain(func(_ *core.Aura, _ *core.Simulation) {
			druid.FeralCharge.Cost.PercentModifier -= 1
			druid.FeralCharge.DefaultCast.GCD = time.Second
		}).ApplyOnExpire(func(_ *core.Aura, _ *core.Simulation) {
			druid.FeralCharge.Cost.PercentModifier += 1
			druid.FeralCharge.DefaultCast.GCD = gcd
		})
	}
}
