package hunter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// Immolation Trap's burn reads attack power once, as it lands: 400 Agility and Strength gained after its first tick leave
// its ticks' base damage where it was (in game the base tick stays fixed for a whole burn).
func TestImmolationTrapReadsAttackPowerOnLanding(t *testing.T) {
	player := &proto.Player{
		Name:               "Trapper",
		Race:               proto.Race_RaceOrc,
		Class:              proto.Class_ClassHunter,
		Equipment:          core.GetGearSet("../../ui/specs/hunter/dps/gear_sets", "p1_melee").GearSet,
		TalentsString:      DefaultSVTalents,
		Spec:               DefaultOptions,
		Rotation:           &proto.APLRotation{},
		DistanceFromTarget: 3,
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	logs := &strings.Builder{}
	sim.Log = func(message string, vals ...interface{}) {
		logs.WriteString(fmt.Sprintf("[%0.2f] "+message+"\n", append([]interface{}{sim.CurrentTime.Seconds()}, vals...)...))
	}
	sim.Reset()
	sim.PrePull()
	hunter := sim.Raid.Parties[0].Players[0].(*Hunter)
	stepTo := func(at time.Duration) {
		sim.AddPendingAction(&core.PendingAction{NextActionAt: at, OnAction: func(*core.Simulation) {}})
		for i := 0; sim.CurrentTime < at; i++ {
			if i > 100000 {
				t.Fatalf("the sim did not reach %v", at)
			}
			sim.Step()
		}
	}
	stepTo(time.Second)
	hunter.WaitUntil(sim, core.NeverExpires) // park the idle rotation, which holds the GCD while it waits
	stepTo(hunter.GCD.ReadyAt())
	if !hunter.GetSpell(core.ActionID{SpellID: immolationTrapRank.ID}).Cast(sim, hunter.CurrentTarget) {
		t.Fatal("Immolation Trap did not cast")
	}
	stepTo(6 * time.Second) // the burn landed at 2.2 s and first ticked at 5.2 s
	hunter.AddStatDynamic(sim, stats.Agility, 400)
	hunter.AddStatDynamic(sim, stats.Strength, 400)
	stepTo(20 * time.Second)

	tick := regexp.MustCompile(`\{SpellID: ` + strconv.Itoa(int(immolationTrapEffectRank.ID)) + `\} \[DEBUG\][^\n]*BaseDamage:([\d.]+)`)
	var bases []float64
	for _, m := range tick.FindAllStringSubmatch(logs.String(), -1) {
		base, _ := strconv.ParseFloat(m[1], 64)
		bases = append(bases, base)
	}
	if len(bases) < 3 {
		t.Fatalf("only %d burn ticks in the log", len(bases))
	}
	for i, b := range bases {
		if b != bases[0] {
			t.Errorf("tick %d base %.2f, the first %.2f: the burn read attack power again after it landed", i+1, b, bases[0])
		}
	}
}
