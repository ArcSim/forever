package hunter

import (
	"os"
	"strings"

	"github.com/wowsims/forever/sim/core"
)

// Arc's measured hawk model (summon_hawk.go): hawk swings follow the hunter's ranged speed, resolve as the hawk's own
// white melee by the level table, and fill a free slot only. Off runs this engine's HawkPet units instead.
var ForeverHawkModel = true

// ARC_OFF with "hawk" in its list turns the model off for a comparison.
func init() {
	for _, name := range strings.Split(os.Getenv("ARC_OFF"), ",") {
		if strings.TrimSpace(name) == "hawk" {
			ForeverHawkModel = false
		}
	}
}

// A trap's hit roll adds Survival Tactics to the hunter's spell hit: its curve gives 5 and 10 (19376), a flat modifier
// on the traps' resist chance.
func (hunter *Hunter) arcTrapHitRoll(spell *core.Spell, roll func()) {
	if !core.ForeverTrapHit {
		roll()
		return
	}
	bonus := spell.BonusHitPercent
	spell.BonusHitPercent = bonus + 5*float64(hunter.Talents.SurvivalTactics)
	roll()
	spell.BonusHitPercent = bonus
}
