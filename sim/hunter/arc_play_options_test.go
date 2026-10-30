package hunter

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	googleproto "google.golang.org/protobuf/proto"
)

// A shipped build with its hunter options changed.
func playBuild(t *testing.T, name string, change func(*proto.HunterOptions)) (*proto.IndividualSimSettings, *proto.Player) {
	t.Helper()
	build := timelineBuild(t, name)
	player := googleproto.Clone(build.Player).(*proto.Player)
	change(player.GetHunter().Options.ClassOptions)
	return build, player
}

// One logged fight of a shipped build with its hunter options changed.
func playFight(t *testing.T, name string, change func(*proto.HunterOptions)) []timelineLine {
	t.Helper()
	build, player := playBuild(t, name, change)
	return timelineFight(t, player, build, build.Encounter)
}

// The build's hunter, its sim set up for a fight but not run.
func playSim(t *testing.T, name string, change func(*proto.HunterOptions)) (*core.Simulation, *Hunter) {
	t.Helper()
	build, player := playBuild(t, name, change)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
		Encounter:  build.Encounter,
		SimOptions: &proto.SimOptions{RandomSeed: 20261026},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Hunter)
}

func runPlaySim(t *testing.T, sim *core.Simulation) {
	t.Helper()
	sim.PrePull()
	for i := 0; !sim.Step(); i++ {
		if i > 10000000 {
			t.Fatal("the fight did not end")
		}
	}
}

var moveReactionRe = regexp.MustCompile(`^Casting ` + regexp.QuoteMeta(core.MoveReactionID.String()) + ` \(Cost = 0\.000, Cast Time = ([0-9.]+)(ms|s|µs), `)

// The move reactions a fight logs, in seconds.
func moveReactions(lines []timelineLine) []float64 {
	var out []float64
	for _, l := range lines {
		if m := moveReactionRe.FindStringSubmatch(l.text); m != nil {
			v := atof(m[1])
			switch m[2] {
			case "ms":
				v /= 1000
			case "µs":
				v /= 1e6
			}
			out = append(out, v)
		}
	}
	return out
}

// Each walk of a fight, from its start to its end.
func walks(lines []timelineLine) [][2]float64 {
	var out [][2]float64
	start := math.NaN()
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l.text, "Casting "+walkID+" (") && l.at >= 0:
			start = l.at
		case l.text == "Aura faded: "+walkID && !math.IsNaN(start):
			out = append(out, [2]float64{start, l.at})
			start = math.NaN()
		}
	}
	return out
}

// The move delay option sets the reaction before every walk: 200 ms waits 200 ms, 0 none, and unset keeps 100 ms.
func TestMoveDelayOptionSetsTheMoveReaction(t *testing.T) {
	for _, c := range []struct {
		delay *int32
		want  float64
	}{{googleproto.Int32(200), 0.2}, {googleproto.Int32(0), 0}, {nil, 0.1}} {
		lines := playFight(t, "dw_kick_weave", func(o *proto.HunterOptions) { o.MoveDelayMs = c.delay })
		reactions := moveReactions(lines)
		if c.want == 0 {
			if len(reactions) != 0 {
				t.Errorf("move delay 0: %d move reactions logged; want none", len(reactions))
			}
			if n := len(walks(lines)); n < 10 {
				t.Errorf("move delay 0: %d walks; want a weave's worth", n)
			}
			continue
		}
		if len(reactions) < 10 {
			t.Fatalf("move delay %.1f s: %d move reactions; want a weave's worth", c.want, len(reactions))
		}
		for _, r := range reactions {
			if math.Abs(r-c.want) > 1e-9 {
				t.Errorf("move delay %.1f s: a move reaction of %.3f s", c.want, r)
				break
			}
		}
	}
}

// The run speed option sets the speed a walk goes at: the weave's 6 yd walks take 6 yd over 8 yd/s, not over 7 (a
// race's own run speed bonus on top of either).
func TestRunSpeedOptionSetsTheWalkSpeed(t *testing.T) {
	_, plain := playSim(t, "dw_kick_weave", func(o *proto.HunterOptions) {})
	_, fast := playSim(t, "dw_kick_weave", func(o *proto.HunterOptions) { o.RunSpeed = 8 })
	if got := fast.GetMovementSpeed() / plain.GetMovementSpeed(); math.Abs(got-8.0/7) > 1e-9 {
		t.Errorf("run speed 8 moves %.4f times as fast as the default; want 8/7", got)
	}
	want, old := 6/fast.GetMovementSpeed(), 6/plain.GetMovementSpeed()
	sixYards := 0
	for _, w := range walks(playFight(t, "dw_kick_weave", func(o *proto.HunterOptions) { o.RunSpeed = 8 })) {
		if d := w[1] - w[0]; math.Abs(d-want) <= 2*logStampSlack {
			sixYards++
		} else if math.Abs(d-old) <= 2*logStampSlack {
			t.Errorf("a walk at %.2f s takes %.3f s, 6 yd at the default speed", w[0], d)
		}
	}
	if sixYards < 5 {
		t.Errorf("%d walks of %.3f s (6 yd at 8 yd/s); want a weave's worth", sixYards, want)
	}
}

// Pro Player and Regular Player: a drawn reaction before each move, none shorter than the mode's least, a walk out that
// stops 2/3 yd past 11 yd, buttons pressed the moment they can be after arriving, and the mode for rotations to read.
func TestWeavePlayOption(t *testing.T) {
	for _, c := range []struct {
		play     proto.HunterOptions_WeavePlay
		min, max float64
	}{{proto.HunterOptions_WeavePerfectPlay, 0.1, 0.2}, {proto.HunterOptions_WeaveNormalPlay, 0.2, 0.3}} {
		set := func(o *proto.HunterOptions) { o.WeavePlay = c.play }

		_, hunter := playSim(t, "dw_kick_weave", set)
		if hunter.ArcPlay.WeavePlay != int32(c.play) {
			t.Errorf("%s: rotations read mode %d", c.play, hunter.ArcPlay.WeavePlay)
		}

		lines := playFight(t, "dw_kick_weave", set)
		reactions := moveReactions(lines)
		if len(reactions) < 10 {
			t.Fatalf("%s: %d move reactions; want a weave's worth", c.play, len(reactions))
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, r := range reactions {
			lo, hi = math.Min(lo, r), math.Max(hi, r)
		}
		if lo < c.min-1e-9 || hi > c.max+1e-9 || hi-lo < 0.05 {
			t.Errorf("%s: move reactions from %.3f to %.3f s; want drawn from %.1f to %.1f s", c.play, lo, hi, c.min, c.max)
		}

		// presses on the GCD wait for nothing after a walk ends: the reaction is on deciding to move (a walk that starts
		// on arrival, decided on the way in, is no end)
		var ends [][2]float64
		all := walks(lines)
		for i, w := range all {
			if i+1 < len(all) && all[i+1][0]-w[1] <= logStampSlack {
				continue
			}
			ends = append(ends, w)
		}
		quick := 0
		for _, l := range lines {
			if !strings.HasPrefix(l.text, "Casting {SpellID") || !strings.Contains(l.text, "GCD = 1") {
				continue
			}
			for _, w := range ends {
				if l.at >= w[1] && l.at < w[1]+c.min-logStampSlack {
					quick++
				}
			}
		}
		if quick < 3 {
			t.Errorf("%s: %d presses inside the least reaction after a walk ends; want buttons pressed the moment they can be", c.play, quick)
		}

		sim, hunter := playSim(t, "dw_kick_weave", set)
		var outs []float64
		hunter.RegisterMovementCallback(func(sim *core.Simulation, position float64, kind core.MovementUpdateType) {
			if kind == core.MovementEnd && sim.CurrentTime > 0 && position > core.MaxMeleeRange {
				outs = append(outs, position)
			}
		})
		runPlaySim(t, sim)
		past := 0
		for _, d := range outs {
			if d >= core.MinRangedRange && d < 11.67 {
				t.Errorf("%s: a walk out stopped at %.2f yd; want 11.67", c.play, d)
				break
			}
			if d == 11.67 {
				past++
			}
		}
		if past < 10 {
			t.Errorf("%s: %d walks out stopped at 11.67 yd; want a weave's worth", c.play, past)
		}
	}
}

// Without the option a melee swing leaves the Auto Shot timer where it was: over a whole fight, no main hand swing
// that fires no shot of its own moves the next Auto Shot.
func TestMeleeLeavesAutoShotWithoutTheResetOption(t *testing.T) {
	sim, hunter := playSim(t, "2h_auto_weave", func(o *proto.HunterOptions) { o.MeleeResetsAutoShot = false })
	aa := &hunter.AutoAttacks
	sim.PrePull()
	swings, moved := 0, 0
	for i := 0; ; i++ {
		if i > 10000000 {
			t.Fatal("the fight did not end")
		}
		swing, shot, nextShot := aa.MainhandPreviousSwing(), aa.PreviousRangedAttack(), aa.NextRangedAttackAt()
		done := sim.Step()
		if aa.MainhandPreviousSwing() != swing && aa.PreviousRangedAttack() == shot {
			swings++
			if aa.NextRangedAttackAt() != nextShot {
				moved++
			}
		}
		if done {
			break
		}
	}
	if swings < 10 {
		t.Fatalf("%d main hand swings; want a weave's worth", swings)
	}
	if moved > 0 {
		t.Errorf("%d of %d main hand swings moved the next Auto Shot; want none without the option", moved, swings)
	}
}

// Linked both ways, every Auto Shot release restarts the main hand: its next swing comes a full swing after the shot.
// Linked one way, a shot leaves the main hand's timer where it was. The off hand keeps its own timer either way.
func TestAutoShotRestartsTheMainHandLinkedBothWays(t *testing.T) {
	for _, both := range []bool{false, true} {
		sim, hunter := playSim(t, "dw_auto_weave_reset", func(o *proto.HunterOptions) {
			o.MeleeResetsAutoShot = true
			o.AutoShotResetsMainHand = both
		})
		aa := &hunter.AutoAttacks
		sim.PrePull()
		shots, restarted, movedOffHand := 0, 0, 0
		for i := 0; ; i++ {
			if i > 10000000 {
				t.Fatal("the fight did not end")
			}
			swing, offSwing, shot := aa.MainhandPreviousSwing(), aa.OffhandPreviousSwing(), aa.PreviousRangedAttack()
			nextSwing, nextOffSwing := aa.MainhandSwingAt(), aa.OffhandSwingAt()
			done := sim.Step()
			if aa.PreviousRangedAttack() != shot && aa.MainhandPreviousSwing() == swing && aa.OffhandPreviousSwing() == offSwing {
				shots++
				if aa.MainhandSwingAt() != nextSwing && aa.MainhandSwingAt() == sim.CurrentTime+aa.MainhandSwingSpeed() {
					restarted++
				}
				if aa.OffhandSwingAt() != nextOffSwing {
					movedOffHand++
				}
			}
			if done {
				break
			}
		}
		if shots < 10 {
			t.Fatalf("both ways %v: %d Auto Shots; want a weave's worth", both, shots)
		}
		if want := map[bool]int{false: 0, true: shots}[both]; restarted != want {
			t.Errorf("both ways %v: %d of %d Auto Shots restarted the main hand; want %d", both, restarted, shots, want)
		}
		if movedOffHand > 0 {
			t.Errorf("both ways %v: %d Auto Shots moved the off hand's timer; want none", both, movedOffHand)
		}
	}
}

// Melee swings restart the Auto Shot timer: each swing starts a full Auto Shot cycle from its instant, and no Auto
// Shot fires before the cycle the last swing started has run out.
func TestMeleeResetsAutoShotOption(t *testing.T) {
	lines := playFight(t, "2h_auto_weave", func(o *proto.HunterOptions) { o.MeleeResetsAutoShot = true })
	shotStarts := map[float64]float64{}
	var swingsAt []float64
	for _, l := range lines {
		m := timerRe.FindStringSubmatch(l.text)
		if m == nil || m[2] == "" {
			continue
		}
		switch m[1] {
		case shotID:
			shotStarts[l.at] = atof(m[3])
		case mhID:
			swingsAt = append(swingsAt, l.at)
		}
	}
	if len(swingsAt) < 10 {
		t.Fatalf("%d main hand swings; want a weave's worth", len(swingsAt))
	}
	for _, s := range swingsAt {
		ready, ok := shotStarts[s]
		if !ok {
			t.Errorf("a main hand swing at %.2f s starts no Auto Shot timer", s)
			continue
		}
		if cycle := ready - s; cycle < 1.5 || cycle > 3.5 {
			t.Errorf("the swing at %.2f s restarts Auto Shot to be ready %.3f s later; want a whole cycle", s, cycle)
		}
	}
	// a shot logs its new timer as it fires, before it completes
	fired := map[float64]bool{}
	for _, l := range lines {
		if l.text == "Completed cast "+shotID {
			fired[l.at] = true
		}
	}
	// standing at range as the timer runs out, the shot fires then, its windup run before it
	lastReady, onTime := math.Inf(-1), 0
	for _, l := range lines {
		if m := timerRe.FindStringSubmatch(l.text); m != nil && m[1] == shotID && m[2] != "" {
			if fired[l.at] && l.at > 0 && l.at < lastReady-logStampSlack {
				t.Errorf("an Auto Shot at %.2f s before its timer ran out at %.2f s", l.at, lastReady)
			}
			if fired[l.at] && math.Abs(l.at-lastReady) <= logStampSlack {
				onTime++
			}
			lastReady = atof(m[3])
		}
	}
	if onTime < 5 {
		t.Errorf("%d Auto Shots fire as the restarted timer runs out; want most of a weave's", onTime)
	}
}

// Shots delay Auto Shot: the first shot or sting cast in an Auto Shot cycle moves that cycle's Auto Shot 0.2 s later,
// as it completes; a second one in the same cycle does not.
func TestShotsDelayAutoShotOption(t *testing.T) {
	for _, on := range []bool{false, true} {
		lines := playFight(t, "bm_ranged", func(o *proto.HunterOptions) { o.ShotsDelayAutoShot = on })
		ready, moved, perCycle := 0.0, 0, 0
		completed := map[float64]bool{}
		for _, l := range lines {
			if strings.HasPrefix(l.text, "Completed cast {SpellID") {
				completed[l.at] = true
			}
			m := timerRe.FindStringSubmatch(l.text)
			if m == nil || m[1] != shotID {
				continue
			}
			if m[2] != "" {
				ready, perCycle = atof(m[3]), 0
				continue
			}
			moved++
			perCycle++
			if !on {
				t.Errorf("off: the Auto Shot timer moved at %.2f s", l.at)
				continue
			}
			if got := atof(m[3]) - ready; math.Abs(got-0.2) > 1e-6 {
				t.Errorf("at %.2f s the Auto Shot moved %.3f s; want 0.2", l.at, got)
			}
			if perCycle > 1 {
				t.Errorf("at %.2f s a second shot in the cycle moved the Auto Shot again", l.at)
			}
			if !completed[l.at] {
				t.Errorf("at %.2f s the Auto Shot moved with no shot completing then", l.at)
			}
			ready = atof(m[3])
		}
		if on && moved < 10 {
			t.Errorf("on: the Auto Shot moved %d times; want a fight's worth", moved)
		}
	}
}

// Mortal Shots' crit bonus covers Auto Shot as it does Arcane Shot: Auto Shot crits at 2.30-2.32x with 5/5 in
// game, as Arcane Shot does.
func TestMortalShotsCoversAutoShot(t *testing.T) {
	_, hunter := playSim(t, "2h_auto_weave", func(*proto.HunterOptions) {})
	if hunter.Talents.MortalShots == 0 {
		t.Fatal("the 2H Auto Weave build has no Mortal Shots")
	}
	auto := hunter.AutoAttacks.RangedAuto().CritMultiplierAdditive
	shot := hunter.GetSpell(core.ActionID{SpellID: arcaneShotRank.ID}).CritMultiplierAdditive
	if auto <= 0 || auto != shot {
		t.Errorf("Auto Shot's crit bonus %.3f, Arcane Shot's %.3f; want Mortal Shots' on both", auto, shot)
	}
}

// A rotation's time to the next Auto Shot counts to the shot, not to the start of its windup. The Kick Weave's Pro and
// Regular Player leads walk back in just before the shot; read to the windup's start, that walk cancelled the shot and
// sent the hunter out again.
func TestRangedTimeToNextCountsToTheShot(t *testing.T) {
	for _, play := range []proto.HunterOptions_WeavePlay{proto.HunterOptions_WeavePerfectPlay, proto.HunterOptions_WeaveNormalPlay} {
		lines := playFight(t, "dw_kick_weave", func(o *proto.HunterOptions) { o.WeavePlay = play })
		windups, shots := 0, 0
		for _, l := range lines {
			switch {
			case strings.HasPrefix(l.text, "Casting "+shotID+" ("):
				windups++
			case l.text == "Completed cast "+shotID:
				shots++
			}
		}
		if shots < 10 || windups != shots {
			t.Errorf("%v: %d Auto Shot windups and %d shots; want every windup to fire", play, windups, shots)
		}
	}
}
