package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/proto"
)

// Hunter's Mark rank 4 gives 143 ranged attack power, the hunter's own cast and the raid's debuff alike.
func TestHuntersMarkGives143RangedAttackPower(t *testing.T) {
	if got := buffs.HuntersMarkValue(0); got != 143 {
		t.Errorf("the raid's Hunter's Mark gives %v ranged attack power, want 143", got)
	}

	sim, hunter := racialHunter(t, proto.Race_RaceHuman)
	target := hunter.CurrentTarget
	before := target.PseudoStats.BonusRangedAttackPower
	hunter.GetSpell(core.ActionID{SpellID: huntersMarkRank.ID}).SkipCastAndApplyEffects(sim, target)
	if got := target.PseudoStats.BonusRangedAttackPower - before; got != 143 {
		t.Errorf("the hunter's own Hunter's Mark gives %v ranged attack power, want 143", got)
	}
}

// The 15% quiver is Harpy Hide Quiver, at the generated 15%.
func TestHarpyHideQuiverIsTheFifteenPercentQuiver(t *testing.T) {
	q, ok := quivers[proto.HunterOptions_HarpyHideQuiver.String()]
	if !ok || q.ItemID != 19319 || q.Haste != 15 {
		t.Errorf("HunterOptions.HarpyHideQuiver is %+v, want Harpy Hide Quiver (19319) at 15%%", q)
	}
	if _, ok := quivers["AncientSinewWrappedLamina"]; ok {
		t.Error("Ancient Sinew Wrapped Lamina is still listed beside Harpy Hide Quiver")
	}
}
