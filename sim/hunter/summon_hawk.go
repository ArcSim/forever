package hunter

import (
	"strconv"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

var summonHawkRank = spellData.SummonHawk.Highest()
var summonHawkBaseDamage = summonHawkRank.DamageEffect().Average()

// The creature the dive leaves behind, up for 18 s.
var hawkSummon = spellData.SummonHawkTriggered.ByID(1293248)

// The tooltip's ${$s1+($rap*($s4/100))}: effect 4 is the share of ranged attack power.
var summonHawkAPShare = summonHawkRank.EffectN(4).Percent()

// A hawk the dive leaves behind: a guardian with a pet's base stats and swing damage, summoned at
// the enemy by the cast.
type HawkPet struct {
	core.Pet
}

func (hawk *HawkPet) GetPet() *core.Pet                   { return &hawk.Pet }
func (hawk *HawkPet) Reset(_ *core.Simulation)            {}
func (hawk *HawkPet) OnEncounterStart(_ *core.Simulation) {}

// A hawk only swings: its rotation sleeps for as long as a hawk can be up.
func (hawk *HawkPet) ExecuteCustomRotation(sim *core.Simulation) {
	hawk.WaitUntil(sim, sim.CurrentTime+hawkSummon.Duration())
}

// What a hawk is, read off level 30 combat logs of 2862 swings at 407 to 1023 owner ranged attack power:
// 5% of the creature health for its level and no armor or attack power of its own; 80% of the owner's
// ranged attack power and 30% of the owner's armor on top. The row that scales a hawk, Hunter Guardian
// Hawk Pet Scaling 01 (1293586), is not in the store, so the swing below is fitted to those logs.
var hawkBaseStats = stats.Stats{
	stats.Health: 0.05 * core.CreatureHealthMaxLevel,
}

// A swing before armor read 0.0815 of the creature dps over the 2.5 s, spanning a third either side, plus
// 0.0179 a point of the owner's ranged attack power: 112 attack power a point of dps at 80% of it. Both
// are fitted (3.3-6.7 plus 0.0179 × ranged attack power at level 30), not stated in the game files, and
// were read with Unleashed Fury 5/5, which reaches the hawks, so the swing here is before it.
var hawkLogDamageMultiplier = spellData.UnleashedFury.MultiplierAt(5)
var hawkSwingAverage = 0.0815 * core.CreatureAutoAttackDpsMaxLevel * 2.5 / hawkLogDamageMultiplier

var hawkSwing = core.Weapon{
	BaseDamageMin:     hawkSwingAverage * (1 - 1.0/3),
	BaseDamageMax:     hawkSwingAverage * (1 + 1.0/3),
	SwingSpeed:        2.5,
	AttackPowerPerDPS: 112 * hawkLogDamageMultiplier,
	MaxRange:          core.MaxMeleeRange,
}

// Effect 3 is the number of hawks up at once. A hawk's swing follows its owner's ranged speed while
// it is up, a quiver's and Quick Shots' included; Berserking's attack speed does not reach it in
// game. Its crit is half the tamed pet's, the higher of melee and ranged (inheritedCrit), with none
// from Agility: 4340 swings with Ferocity 5/5 read 10 plus half the owner's crit against targets 2
// levels below to 3 above it, and 0-Ferocity logs half the tamed pet's. Its melee hit and expertise are
// the tamed pet's: 1% hit took a hawk from 1 miss in 32 swings to none in 515 against a target 8
// levels below it.
func (hunter *Hunter) newHawks() []*HawkPet {
	if !hunter.Talents.SummonHawk || ForeverHawkModel {
		return nil
	}

	hawks := make([]*HawkPet, int(summonHawkRank.EffectN(3).BaseValue()))
	for i := range hawks {
		hawk := &HawkPet{Pet: core.NewPet(core.PetConfig{
			Name:      "Hawk " + strconv.Itoa(i+1),
			Owner:     &hunter.Character,
			BaseStats: hawkBaseStats,
			StatInheritance: func(ownerStats stats.Stats) stats.Stats {
				return stats.Stats{
					stats.Armor:               ownerStats[stats.Armor] * 0.3,
					stats.AttackPower:         ownerStats[stats.RangedAttackPower] * 0.8,
					stats.PhysicalCritPercent: hunter.inheritedCrit(ownerStats) * 0.5,
					stats.PhysicalHitPercent:  inheritedHit(ownerStats),
					stats.ExpertisePercent:    ownerStats[stats.ExpertisePercent],
				}
			},
			IsGuardian:                       true,
			IsDynamic:                        true,
			HasDynamicRangedSpeedInheritance: true,
			StartsInMeleeRange:               true,
		})}
		hawk.EnableAutoAttacks(hawk, core.AutoAttackOptions{
			MainHand:       hawkSwing,
			AutoSwingMelee: true,
		})
		// The damage done is the owner's as the hawk is summoned, on top of the hawk's own.
		inherited := 1.0
		hawk.OnPetEnable = func(sim *core.Simulation) {
			inherited = hunter.PseudoStats.DamageDealtMultiplier
			hawk.PseudoStats.DamageDealtMultiplier *= inherited
		}
		hawk.OnPetDisable = func(sim *core.Simulation) {
			hawk.PseudoStats.DamageDealtMultiplier /= inherited
		}
		hunter.AddPet(hawk)
		hawks[i] = hawk
	}
	return hawks
}

// The tamed pet and the hawks: what the talents that name pets and hawks reach.
func (hunter *Hunter) pets() []*core.Pet {
	var pets []*core.Pet
	if hunter.Pet != nil {
		pets = append(pets, &hunter.Pet.Pet)
	}
	for _, hawk := range hunter.Hawks {
		pets = append(pets, &hawk.Pet)
	}
	return pets
}

// The dive's damage, then a hawk for 18 s. The hawks take turns, so one past the count replaces
// the one with the least time left.
func (hunter *Hunter) registerSummonHawk() {
	if !hunter.Talents.SummonHawk {
		return
	}

	config := spelldata.SpellConfig(&hunter.Unit, summonHawkRank, spelldata.Melee(core.ProcMaskRangedSpecial))
	summonForeverHawk := hunter.registerForeverHawks()
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseDamage := summonHawkBaseDamage + summonHawkAPShare*spell.RangedAttackPower(target)
		result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedCritOnly)
		spell.DealDamageAfterTravel(sim, result)

		if summonForeverHawk != nil {
			summonForeverHawk(sim, target)
			return
		}

		hawk := hunter.Hawks[hunter.nextHawk]
		hunter.nextHawk = (hunter.nextHawk + 1) % len(hunter.Hawks)
		if hawk.IsEnabled() {
			hawk.Disable(sim)
		}
		hawk.EnableWithTimeout(sim, hawk, hawkSummon.Duration())
	}

	hunter.RegisterSpell(config)
}

// The measured hawk (ForeverHawkModel), from level 30 logs: a swing every 2.57 s over the hunter's ranged speed (2.27 s
// with a 13% quiver; Rapid Fire and Quick Shots reach it, the melee Quick Strikes does not). Its attack power is 80% of
// the hunter's ranged attack power (264 of 264 swings beside an owner swing at 0.800) and its crit half the hunter's
// plus Ferocity (an owner's pet critting 22.2% and hawk 16.2% solve to 12% owner crit and 10% Ferocity).
const (
	foreverHawkSwingSpeed = 2570 * time.Millisecond
	// 0.0815 of the level 60 creature dps over 2.5 s with Unleashed Fury 5/5's 20% taken out, a third either side,
	// plus 2.5 / 134.4 a point of the hawk's attack power (a fit of 2,862 swings at 407 to 1,023 ranged attack power)
	foreverHawkSwingBaseDamage = 0.0815 * 60.45 * 2.5 / 1.2
	foreverHawkSwingPerHawkAP  = 2.5 / 134.4
	foreverHawkAPShare         = 0.8
	foreverHawkOwnerCritShare  = 0.5
	foreverHawkCritPerFerocity = 0.02
	foreverHawkUnleashedFury   = 0.03
)

// AttackSpeedMultiplier is folded in: the measured model reads one ranged speed that every attack speed effect moves.
func (hunter *Hunter) foreverHawkSwingInterval() time.Duration {
	return time.Duration(float64(foreverHawkSwingSpeed) / (hunter.PseudoStats.RangedSpeedMultiplier * hunter.PseudoStats.AttackSpeedMultiplier))
}

// one swing before Unleashed Fury, armor and outcome; roll in [0, 1) spreads the base a third either side
func foreverHawkSwingBase(roll, rangedAP float64) float64 {
	return foreverHawkSwingBaseDamage*(2.0/3+roll*2.0/3) + foreverHawkSwingPerHawkAP*foreverHawkAPShare*rangedAP
}

// half the hunter's crit (melee and ranged differ only by a main-hand stone, so the melee side) plus Ferocity
func (hunter *Hunter) foreverHawkCritChance() float64 {
	owner := (hunter.GetStat(stats.PhysicalCritPercent) + max(hunter.ScopedPercents(core.ProcMaskMeleeMHAuto).Crit, 0)) / 100
	return foreverHawkOwnerCritShare*owner + foreverHawkCritPerFerocity*float64(hunter.Talents.Ferocity)
}

// The measured hawks as the hunter's own swings: a dive fills the first free slot for 18 s, none when every slot is
// up; the hawk swings one interval after it arrives. The returned summon is nil with the model off.
func (hunter *Hunter) registerForeverHawks() func(sim *core.Simulation, target *core.Unit) {
	if !ForeverHawkModel {
		return nil
	}

	slots := int(summonHawkRank.EffectN(3).BaseValue())
	duration := hawkSummon.Duration()
	expireAt := make([]time.Duration, slots)
	alive := make([]bool, slots)
	generation := make([]int, slots)
	var swingTarget *core.Unit

	// white melee by the target's level from the hunter's level with no weapon skill: miss, dodge, glancing, then the
	// hawk's own crit for double damage; the hawk's hit and expertise are not modeled
	var hawkSwing *core.Spell
	hawkOutcome := func(sim *core.Simulation, result *core.SpellResult, _ *core.AttackTable) {
		metrics := &hawkSwing.SpellMetrics[result.Target.UnitIndex]
		diff := float64(result.Target.Level*5 - hunter.Level*5)
		miss := 0.05 + diff*0.001
		if diff > 10 {
			miss = 0.05 + diff*0.002
		}
		dodge := 0.05 + diff*0.001
		glance := 0.1 + diff*0.02
		hawkCrit := hunter.foreverHawkCritChance()
		roll := sim.RandomFloat("Hawk Hit Table")
		switch {
		case roll < miss:
			result.Outcome = core.OutcomeMiss
			result.Damage = 0
			metrics.Misses++
		case roll < miss+dodge:
			result.Outcome = core.OutcomeDodge
			result.Damage = 0
			metrics.Dodges++
		case roll < miss+dodge+glance:
			result.Outcome = core.OutcomeGlance
			low := max(min(1.3-0.05*diff, 0.91), 0.01)
			high := max(min(1.2-0.03*diff, 0.99), 0.2)
			result.Damage *= low + sim.RandomFloat("Hawk Glancing Penalty")*(high-low)
			metrics.Glances++
		case roll < miss+dodge+glance+hawkCrit:
			result.Outcome = core.OutcomeCrit
			result.Damage *= 2
			metrics.Crits++
		default:
			result.Outcome = core.OutcomeHit
			metrics.Hits++
		}
	}

	hawkSwing = hunter.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: summonHawkRank.ID}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		// the hawk's attack sets off none of the hunter's on-hit effects
		ProcMask: core.ProcMaskEmpty,
		// a guardian's attack: the hunter's own damage modifiers do not reach it (the flag drops the spell's own
		// multiplier too, so Unleashed Fury goes on the base)
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagIgnoreAttackerModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			if swingTarget == nil {
				return
			}
			unleashedFury := 1 + foreverHawkUnleashedFury*float64(hunter.Talents.UnleashedFury)
			// the fit read the hunter's own ranged attack power: the target's Hunter's Mark bonus is not in it
			rangedAP := spell.RangedAttackPower(swingTarget) - swingTarget.PseudoStats.BonusRangedAttackPower
			base := unleashedFury * foreverHawkSwingBase(sim.RandomFloat("Hawk Damage"), rangedAP)
			// CalcDamage adds the hunter's flat physical damage, and the target's flat physical damage taken after
			// armor, to every physical hit; the measured swing takes neither, so both come off here
			attackTable := hunter.AttackTables[swingTarget.UnitIndex]
			base -= spell.BonusDamage(attackTable) + swingTarget.PseudoStats.BonusPhysicalDamageTaken/attackTable.GetArmorDamageModifier(spell)
			spell.CalcAndDealDamage(sim, swingTarget, base, hawkOutcome)
		},
	})

	// the stacks count the hawks up, for rotations that read them
	hawkAura := hunter.RegisterAura(core.Aura{
		Label:     "Summon Hawk",
		ActionID:  core.ActionID{SpellID: summonHawkRank.ID}.WithTag(2),
		Duration:  core.NeverExpires,
		MaxStacks: int32(slots),
		// every iteration starts with no hawks out (old expiry times would block the slots all fight)
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			for i := range expireAt {
				expireAt[i] = 0
				alive[i] = false
				generation[i]++
			}
		},
	})

	// a hawk's stack goes the moment its time runs out, not at its next swing, so a rotation counting the hawks sees it
	// gone
	expire := func(sim *core.Simulation, slot int, gen int) {
		if generation[slot] != gen || !alive[slot] {
			return
		}
		alive[slot] = false
		if hawkAura.IsActive() {
			hawkAura.RemoveStack(sim)
			if hawkAura.GetStacks() == 0 {
				hawkAura.Deactivate(sim)
			}
		}
	}

	hunter.ArcPlay.HawkTimeLeft = func(sim *core.Simulation, pick proto.HawkPick) time.Duration {
		timeLeft := func(slot int) time.Duration {
			if slot >= slots || !alive[slot] {
				return 0
			}
			return max(expireAt[slot]-sim.CurrentTime, 0)
		}
		switch pick {
		case proto.HawkPick_HawkPickSlot1:
			return timeLeft(0)
		case proto.HawkPick_HawkPickSlot2:
			return timeLeft(1)
		}
		last := pick == proto.HawkPick_HawkPickLast
		left, found := time.Duration(0), false
		for i := range expireAt {
			if !alive[i] {
				continue
			}
			if remaining := timeLeft(i); !found || (last && remaining > left) || (!last && remaining < left) {
				left, found = remaining, true
			}
		}
		return left
	}

	return func(sim *core.Simulation, target *core.Unit) {
		swingTarget = target

		for i := 0; i < slots; i++ {
			if expireAt[i] > sim.CurrentTime {
				continue
			}
			slot := i
			expireAt[slot] = sim.CurrentTime + duration
			// a re-summon can land before the old hawk's swing timer saw it expire: the new hawk takes the slot and
			// its stack, the old timer just stops
			generation[slot]++
			myGeneration := generation[slot]
			if !alive[slot] {
				alive[slot] = true
				if !hawkAura.IsActive() {
					hawkAura.Activate(sim)
				}
				hawkAura.AddStack(sim)
			}
			sim.AddPendingAction(&core.PendingAction{
				NextActionAt: expireAt[slot],
				Priority:     core.ActionPriorityDOT,
				OnAction: func(sim *core.Simulation) {
					expire(sim, slot, myGeneration)
				},
			})

			var pa *core.PendingAction
			pa = &core.PendingAction{
				NextActionAt: sim.CurrentTime + hunter.foreverHawkSwingInterval(),
				Priority:     core.ActionPriorityDOT,
				OnAction: func(sim *core.Simulation) {
					if generation[slot] != myGeneration {
						return
					}
					if sim.CurrentTime > expireAt[slot] || swingTarget == nil {
						expireAt[slot] = 0
						expire(sim, slot, myGeneration)
						return
					}
					hawkSwing.Cast(sim, swingTarget)
					pa.NextActionAt = sim.CurrentTime + hunter.foreverHawkSwingInterval()
					sim.AddPendingAction(pa)
				},
			}
			sim.AddPendingAction(pa)
			break
		}
	}
}
