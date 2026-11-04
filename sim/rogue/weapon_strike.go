package rogue

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The game takes a strike's weapon at its normalized speed only when the strike carries a normalized
// weapon damage effect; a strike with only a weapon percent uses the weapon's own speed.
func normalizesWeapon(s *spelldata.Spell) bool {
	for _, e := range s.Effects {
		if e.Type == dbcenums.E_NORMALIZED_WEAPON_DMG {
			return true
		}
	}
	return false
}

func mhStrikeDamage(s *spelldata.Spell, unit *core.Unit, sim *core.Simulation, attackPower float64) float64 {
	if normalizesWeapon(s) {
		return unit.MHNormalizedWeaponDamage(sim, attackPower)
	}
	return unit.MHWeaponDamage(sim, attackPower)
}

func ohStrikeDamage(s *spelldata.Spell, unit *core.Unit, sim *core.Simulation, attackPower float64) float64 {
	if normalizesWeapon(s) {
		return unit.OHNormalizedWeaponDamage(sim, attackPower)
	}
	return unit.OHWeaponDamage(sim, attackPower)
}
