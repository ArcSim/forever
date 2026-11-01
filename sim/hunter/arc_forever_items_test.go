package hunter

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
	googleproto "google.golang.org/protobuf/proto"
)

// Blackhand's Breadth as Forever's data has it: the equip 1% crit (aura 52) is melee and ranged only, and the use
// (+5% crit for 20 s, 5 min cooldown) locks the trinket category 1141 for 20 s, so the builds' Adaptive Combat
// Assistant waits for it.
func TestBlackhandsBreadth(t *testing.T) {
	crit := func(wearIt bool) [3]float64 {
		build := timelineBuild(t, "2h_auto_weave")
		player := googleproto.Clone(build.Player).(*proto.Player)
		worn := false
		for _, item := range player.Equipment.Items {
			if item.GetId() == 13965 {
				worn = true
				if !wearIt {
					item.Id = 0
				}
			}
		}
		if !worn {
			t.Fatal("2H Auto Weave no longer wears Blackhand's Breadth")
		}
		result := core.ComputeStats(&proto.ComputeStatsRequest{
			Raid:      core.SinglePlayerRaidProto(player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
			Encounter: build.Encounter,
		})
		sheet := result.RaidStats.Parties[0].Players[0].FinalStats.PseudoStats
		return [3]float64{sheet[proto.PseudoStat_PseudoStatMeleeCritPercent], sheet[proto.PseudoStat_PseudoStatRangedCritPercent],
			sheet[proto.PseudoStat_PseudoStatSpellCritPercent]}
	}
	with, without := crit(true), crit(false)
	for i, want := range [3]float64{1, 1, 0} {
		if got := with[i] - without[i]; math.Abs(got-want) > 1e-6 {
			t.Errorf("worn, melee / ranged / spell crit go %v -> %v, want +%v each", without, with, [3]float64{1, 1, 0})
			break
		}
	}

	sim, hunter := playSim(t, "2h_auto_weave", func(*proto.HunterOptions) {})
	breadth := hunter.GetSpell(core.ActionID{ItemID: 13965})
	assistant := hunter.GetSpell(core.ActionID{ItemID: 272437})
	if breadth == nil || assistant == nil {
		t.Fatal("the build's two on-use trinkets register no spells")
	}
	if breadth.CD.Duration != 5*time.Minute || breadth.SharedCD.Timer != hunter.GetOrInitSpellCategoryTimer(1141) ||
		breadth.SharedCD.Duration != 20*time.Second {
		t.Errorf("cooldown %v, category lock %v, want 5m0s and 20s on category 1141", breadth.CD.Duration, breadth.SharedCD.Duration)
	}
	before := hunter.GetStat(stats.CritRating)
	if !breadth.Cast(sim, hunter.CurrentTarget) {
		t.Fatal("Blackhand's Breadth could not be used")
	}
	if got := hunter.GetStat(stats.CritRating) - before; got != 5*core.CritRatingPerCritPercent {
		t.Errorf("the use gave %v crit rating, want %v", got, 5*core.CritRatingPerCritPercent)
	}
	if aura := hunter.GetAura("Blackhand's Breadth"); aura == nil || !aura.IsActive() || aura.Duration != 20*time.Second {
		t.Errorf("the use's aura is %v, want active for 20s", aura)
	}
	if assistant.IsReady(sim) {
		t.Error("Adaptive Combat Assistant is ready right after Blackhand's Breadth, want the category locked")
	}
}
