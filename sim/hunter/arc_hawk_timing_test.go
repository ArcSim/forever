package hunter

import (
	"os"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"google.golang.org/protobuf/encoding/protojson"
)

// The held Hawk Weave build's hunter (Summon Hawk talented) with no rotation, so only the test summons hawks.
func hawkSim(t *testing.T) (*core.Simulation, *Hunter) {
	t.Helper()
	raw, err := os.ReadFile("../../ui/specs/hunter/dps/held_builds/hawk_weave.build.json")
	if err != nil {
		t.Fatal(err)
	}
	build := &proto.IndividualSimSettings{}
	if err := protojson.Unmarshal(raw, build); err != nil {
		t.Fatal(err)
	}
	player := build.Player
	player.Rotation = &proto.APLRotation{}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
		Encounter:  build.Encounter,
		SimOptions: &proto.SimOptions{RandomSeed: 20261029},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

func stepTo(t *testing.T, sim *core.Simulation, at time.Duration) {
	t.Helper()
	for i := 0; sim.CurrentTime < at; i++ {
		if i > 1000000 || sim.Step() {
			t.Fatalf("the fight ended before %v", at)
		}
	}
}

// A hawk leaves when its 18 s run out: its stack goes then, not at the hawk's next swing past it, so a rotation that
// counts the hawks sees the right number.
func TestHawkStacksDropWhenEachHawkExpires(t *testing.T) {
	sim, hunter := hawkSim(t)
	summon := hunter.GetSpell(core.ActionID{SpellID: summonHawkRank.ID})
	hawks := hunter.GetAuraByID(core.ActionID{SpellID: summonHawkRank.ID}.WithTag(2))
	if summon == nil || hawks == nil {
		t.Fatal("the hunter has no Summon Hawk")
	}
	duration := hawkSummon.Duration()

	stepTo(t, sim, time.Second)
	summon.SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
	first := sim.CurrentTime
	stepTo(t, sim, first+5*time.Second)
	summon.SkipCastAndApplyEffects(sim, hunter.CurrentTarget)
	second := sim.CurrentTime

	// a third hawk a second after the first leaves takes the slot it freed, slot 1
	third := first + duration + time.Second
	sim.AddPendingAction(&core.PendingAction{
		NextActionAt: third,
		OnAction:     func(sim *core.Simulation) { summon.SkipCastAndApplyEffects(sim, hunter.CurrentTarget) },
	})

	for _, tc := range []struct {
		at                       time.Duration
		stacks                   int32
		next, last, slot1, slot2 time.Duration
	}{
		{second + time.Second, 2, first + duration - second - time.Second, duration - time.Second, first + duration - second - time.Second, duration - time.Second},
		{first + duration - time.Millisecond, 2, time.Millisecond, second - first + time.Millisecond, time.Millisecond, second - first + time.Millisecond},
		// the first hawk's last instant, read before it leaves: none left on it
		{first + duration, 2, 0, second - first, 0, second - first},
		{first + duration + time.Millisecond, 1, second - first - time.Millisecond, second - first - time.Millisecond, 0, second - first - time.Millisecond},
		{third + time.Second, 2, second - third - time.Second + duration, duration - time.Second, duration - time.Second, second - third - time.Second + duration},
		{second + duration + time.Millisecond, 1, third + duration - second - duration - time.Millisecond, third - second - time.Millisecond, third - second - time.Millisecond, 0},
		{third + duration + time.Millisecond, 0, 0, 0, 0, 0},
	} {
		// read at exactly that moment, ahead of anything else then
		sim.AddPendingAction(&core.PendingAction{
			NextActionAt: tc.at,
			Priority:     core.ActionPriorityPrePull,
			OnAction: func(sim *core.Simulation) {
				if got := hawks.GetStacks(); got != tc.stacks {
					t.Errorf("at %v (hawks summoned at %v, %v and %v, %v each): %d hawks counted, want %d", tc.at, first, second, third, duration, got, tc.stacks)
				}
				for _, read := range []struct {
					pick proto.HawkPick
					want time.Duration
				}{
					{proto.HawkPick_HawkPickNext, tc.next},
					{proto.HawkPick_HawkPickLast, tc.last},
					{proto.HawkPick_HawkPickSlot1, tc.slot1},
					{proto.HawkPick_HawkPickSlot2, tc.slot2},
				} {
					if got := hunter.ArcPlay.HawkTimeLeft(sim, read.pick); got != read.want {
						t.Errorf("at %v: %s has %v left, want %v", tc.at, read.pick, got, read.want)
					}
				}
			},
		})
	}
	stepTo(t, sim, third+duration+time.Second)
}
