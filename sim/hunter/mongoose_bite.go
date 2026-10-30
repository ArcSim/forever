package hunter

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var mongooseBiteRank = spellData.MongooseBite.Highest()

// The window the hunter's own dodge opens: the aura Defensive State fires when the hunter dodges an attack.
var mongooseBiteWindow = spellData.DefensiveStateTriggered.Highest()

func (hunter *Hunter) registerMongooseBite() {
	hunter.MongooseBiteAura = hunter.RegisterAura(spelldata.AuraConfig(mongooseBiteWindow))

	hunter.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Mongoose Bite - Trigger",
		TriggerImmediately: true,
		// Defensive State (13358) hears melee swings and abilities taken, not ranged attacks.
		ProcMask: core.ProcMaskMelee,
		Outcome:  core.OutcomeDodge,
		Callback: core.CallbackOnSpellHitTaken,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			hunter.MongooseBiteAura.Activate(sim)
		},
	})

	exposePrey, spendExposePrey := hunter.registerExposePrey()

	config := spelldata.SpellConfig(&hunter.Unit, mongooseBiteRank, spelldata.Melee(core.ProcMaskMeleeMHSpecial))

	config.ExtraCastCondition = func(sim *core.Simulation, target *core.Unit) bool {
		return hunter.MongooseBiteAura.IsActive() || exposePrey.IsActive()
	}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := mongooseBiteRank.WeaponStrike(sim, spell, hunter.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target)), 0)
		spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		// TODO: Ingame test needed: which window a bite spends when both are up.
		hunter.MongooseBiteAura.Deactivate(sim)
		spendExposePrey(sim)
	}

	hunter.RegisterSpell(config)
}

// The window Expose Prey opens: the aura its proc casts.
var exposePreyWindow = spellData.ExposePreyTriggered.Highest()

// A marker is a blip: long enough to show on the timeline, short enough to read as one moment.
const exposePreyMarkerDuration = 100 * time.Millisecond

// Expose Prey opens its own Mongoose Bite window off any attack on a target carrying Hunter's Mark.
// The rate is the effect ladder the tooltip's $m1 names, 5/10 by rank, and the mark is the caller's
// condition. Without the talent the window is still registered and never opens, so a rotation that
// checks it reads it as down rather than dropping the check. The second return spends the window, as
// a Mongoose Bite does.
func (hunter *Hunter) registerExposePrey() (*core.Aura, func(*core.Simulation)) {
	config := spelldata.AuraConfig(exposePreyWindow)
	if hunter.Talents.ExposePrey == 0 {
		window := hunter.RegisterAura(config)
		return window, window.Deactivate
	}

	// Short markers under the window's own id, so the buff table and the timeline show what became of each
	// proc: a bite spent it, it ran out unused, or a new proc refreshed it while it was up.
	var used, expired *core.Aura
	spent := false
	config.OnExpire = func(aura *core.Aura, sim *core.Simulation) {
		// the fight's end takes it down too, and that wastes nothing
		if spent {
			used.Activate(sim)
		} else if sim.CurrentTime < sim.Duration {
			expired.Activate(sim)
		}
		spent = false
	}
	window := hunter.RegisterAura(config)
	marker := func(tag int32, label string) *core.Aura {
		return hunter.RegisterAura(core.Aura{Label: label, ActionID: window.ActionID.WithTag(tag), Duration: exposePreyMarkerDuration})
	}
	used = marker(1, "Expose Prey Used")
	expired = marker(2, "Expose Prey Expired")
	refreshed := marker(3, "Expose Prey Refreshed")

	trigger := spelldata.ProcTrigger(&hunter.Character, spellData.ExposePrey.Rank(hunter.Talents.ExposePrey),
		func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if window.IsActive() {
				refreshed.Activate(sim)
			}
			window.Activate(sim)
		})
	trigger.Name = "Expose Prey - Trigger"
	trigger.ExtraCondition = func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) bool {
		return result.Target.HasActiveAuraWithTag(buffs.HuntersMarkCategory)
	}

	hunter.MakeProcTriggerAura(trigger)
	return window, func(sim *core.Simulation) {
		if window.IsActive() {
			spent = true
			window.Deactivate(sim)
		}
	}
}
