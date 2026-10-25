package mage

import (
	"math"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/spelldata"
)

var blinkRank = spellData.Blink.Highest()

// Blink
// https://www.wowhead.com/forever/spell=1953
//
// The leap's radius is how far the mage moves toward the target, with no travel time. From closer
// than that the mage passes through the target and lands the rest of the way past it. The
// mechanic immunities (effects 1 and 2) do nothing against a boss the sim fights.
func (mage *Mage) registerBlinkSpell() {
	leap := float64(blinkRank.FindEffect(dbcenums.E_LEAP, 0, 0).RadiusMax)

	config := spelldata.SpellConfig(&mage.Unit, blinkRank, spelldata.Flags(core.SpellFlagAPL|core.SpellFlagMovesToTarget))
	config.ClassSpellMask = MageSpellBlink

	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
		mage.MoveInstantlyTo(math.Abs(mage.DistanceFromTarget-leap), sim)
	}

	mage.RegisterSpell(config)
}
