package buffs

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The generated buffs as they are: Forever has no talents that improve raid buffs, and the client values already
// hold Classic's improved share (Gift of the Wild 16 stats, Grace of Air 89).
func applyForeverBuffs(char *core.Character, raid *proto.RaidBuffs, party *proto.PartyBuffs, individual *proto.IndividualBuffs) {
	applyGeneratedBuffs(char, raid, party, individual)
}

// Amounts the database rows get wrong (core.ForeverMarkAndTrueshot): Hunter's Mark rank 4 is 143 in the newer client
// (the ranks are Classic's x1.3), 71 in these rows.
var foreverValues = map[*Meta]float64{huntersMarkMeta: 143}

// The factor that brings a buff's amounts to its value in foreverValues, 1 for the rest.
func foreverValueScale(m *Meta) float64 {
	target, ok := foreverValues[m]
	if !ok || !core.ForeverMarkAndTrueshot {
		return 1
	}
	applied := spelldata.DryRun(m.Spell, spelldata.RaidBuffOptions(m.SkipAuras, m.FullComboPoints)...).Applied
	if len(applied) == 0 || applied[0].Value == 0 {
		return 1
	}
	return target / applied[0].Value
}
