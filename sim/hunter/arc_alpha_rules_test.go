package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/stats"
)

// Forever rules the alpha engine had that the first port of the beta lost, and alpha rules the evidence turned down.

// Furious Howl at the client row's full value, on the wolf and the hunter both. The alpha gave the hunter alone 40%
// less (from Oct 15 notes nobody saved); Arc's logs show the pet buffing itself and a party member, and the pet's
// attack power rising 32 to 33 from rank 2's 34 on Oct 12 and on Oct 15 after the notes. The row resolves to 83 at
// 60 (82 + 0.4 a level, truncated).
func TestArcFuriousHowlIsTheFullRowOnPetAndHunter(t *testing.T) {
	sim, hunter := newTalentSim("", proto.HunterOptions_Wolf)
	for _, unit := range []*core.Character{&hunter.Character, &hunter.Pet.Character} {
		howl := unit.GetAura("Furious Howl")
		if howl == nil {
			t.Fatalf("no Furious Howl aura on %s", unit.Label)
		}
		before := unit.GetStat(stats.AttackPower)
		howl.Activate(sim)
		if got := unit.GetStat(stats.AttackPower) - before; math.Abs(got-83) > 1e-6 {
			t.Errorf("Furious Howl gives %s +%.2f attack power, want 83", unit.Label, got)
		}
	}
}

// Skysight's run speed (Windshaper Skyborne, 1259688) is A_MOD_INCREASE_SPEED, as Aspect of the Cheetah is: only the
// highest of them counts, so under the aspect's 30% it adds nothing, and its 15% shows again when the aspect drops.
func TestArcSkysightDoesNotStackWithCheetah(t *testing.T) {
	sim, hunter := racialHunter(t, proto.Race_RaceWindshaperSkyborne)
	skysight := hunter.GetAuraByID(core.ActionID{SpellID: 1259688})
	cheetah := hunter.GetAuraByID(core.ActionID{SpellID: aspectOfTheCheetahRank.ID})
	if skysight == nil || cheetah == nil {
		t.Fatal("no Skysight or Aspect of the Cheetah aura")
	}
	base := hunter.GetMovementSpeed()
	check := func(what string, want float64) {
		t.Helper()
		if got := hunter.GetMovementSpeed() / base; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: run speed x%.4f, want x%.4f", what, got, want)
		}
	}
	skysight.Activate(sim)
	check("Skysight", 1.15)
	cheetah.Activate(sim)
	check("Skysight under Cheetah", 1+aspectOfTheCheetahRank.EffectN(1).Percent())
	cheetah.Deactivate(sim)
	check("Skysight after Cheetah drops", 1.15)
}

// Read Leyline (High Order Skyborne, 1270842) raises spirit regeneration 50%: its A_MOD_POWER_REGEN_PERCENT leaves
// mana per 5 alone.
func TestArcReadLeylineRaisesSpiritRegenOnly(t *testing.T) {
	sim, hunter := racialHunter(t, proto.Race_RaceHighOrderSkyborne)
	energized := hunter.GetAuraByID(core.ActionID{SpellID: 1270842})
	if energized == nil {
		t.Fatal("no Read Leyline aura")
	}
	hunter.AddStatsDynamic(sim, stats.Stats{stats.MP5: 50})
	before := hunter.ManaRegenPerSecondWhileNotCasting()
	spirit := hunter.SpiritManaRegenPerSecond() * hunter.PseudoStats.SpiritRegenMultiplier
	energized.Activate(sim)
	if got := hunter.ManaRegenPerSecondWhileNotCasting() - before; math.Abs(got-0.5*spirit) > 1e-9 {
		t.Errorf("Read Leyline adds %.4f mana a second, want %.4f (half the spirit part, none of the 10 a second from mana per 5)", got, 0.5*spirit)
	}
	energized.Deactivate(sim)
	if got := hunter.ManaRegenPerSecondWhileNotCasting(); math.Abs(got-before) > 1e-9 {
		t.Errorf("after Read Leyline: %.4f mana a second, want %.4f", got, before)
	}
}
