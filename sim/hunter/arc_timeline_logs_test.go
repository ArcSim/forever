package hunter

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// The log stamps each line to 0.01 s: two events in the same instant can read up to this far apart.
const logStampSlack = 0.006

type timelineLine struct {
	at   float64
	text string
}

// A shipped build, or one held off the page that the tests still play (the BM builds).
func timelineBuild(t *testing.T, name string) *proto.IndividualSimSettings {
	t.Helper()
	raw, err := os.ReadFile("../../ui/specs/hunter/dps/builds/" + name + ".build.json")
	if os.IsNotExist(err) {
		raw, err = os.ReadFile("../../ui/specs/hunter/dps/held_builds/" + name + ".build.json")
	}
	if err != nil {
		t.Fatal(err)
	}
	build := &proto.IndividualSimSettings{}
	if err := protojson.Unmarshal(raw, build); err != nil {
		t.Fatalf("build %s: %v", name, err)
	}
	return build
}

// The player's own log lines (not the pet's or the target's) of one logged fight.
func timelineFight(t *testing.T, player *proto.Player, build *proto.IndividualSimSettings, encounter *proto.Encounter) []timelineLine {
	t.Helper()
	res := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 20261026, Debug: true},
	})
	if res.Error != nil {
		t.Fatalf("sim error: %s", res.Error.Message)
	}
	lineRe := regexp.MustCompile(`^\[(-?[0-9.]+)\] \[[^\]-]*\(#1\)\] (.*)$`)
	var lines []timelineLine
	for _, l := range strings.Split(res.Logs, "\n") {
		if m := lineRe.FindStringSubmatch(l); m != nil {
			at, _ := strconv.ParseFloat(m[1], 64)
			lines = append(lines, timelineLine{at, m[2]})
		}
	}
	return lines
}

func timelineBuildFight(t *testing.T, name string) []timelineLine {
	build := timelineBuild(t, name)
	return timelineFight(t, build.Player, build, build.Encounter)
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

var (
	mhID   = core.ActionID{OtherID: proto.OtherAction_OtherActionAttack}.WithHand(proto.ItemSlot_ItemSlotMainHand).String()
	ohID   = core.ActionID{OtherID: proto.OtherAction_OtherActionAttack}.WithHand(proto.ItemSlot_ItemSlotOffHand).String()
	shotID = core.ActionID{OtherID: proto.OtherAction_OtherActionShoot}.String()
	walkID = core.ActionID{OtherID: proto.OtherAction_OtherActionMove}.String()

	timerRe = regexp.MustCompile(`^Swing timer (\{[^}]*\}) (started, )?ready at (-?[0-9.]+)s$`)
	waitRe  = regexp.MustCompile(`^Swing wait (\{[^}]*\}) (\w+) from (-?[0-9.]+)s to (-?[0-9.]+)s(?:, steps from (-?[0-9.]+)s)?(?:, stopped at (-?[0-9.]+)s)?$`)
)

type swingWait struct {
	at                      float64
	weapon, reason          string
	from, to                float64
	origin, stoppedAt       float64
	hasOrigin, hasStoppedAt bool
}

func swingWaits(lines []timelineLine) []swingWait {
	var waits []swingWait
	for _, l := range lines {
		if m := waitRe.FindStringSubmatch(l.text); m != nil {
			w := swingWait{at: l.at, weapon: m[1], reason: m[2], from: atof(m[3]), to: atof(m[4])}
			w.hasOrigin, w.hasStoppedAt = m[5] != "", m[6] != ""
			w.origin, w.stoppedAt = atof(m[5]), atof(m[6])
			waits = append(waits, w)
		}
	}
	return waits
}

// When each weapon swung: a main-hand swing may go off as a queued Raptor Strike, and an Auto Shot goes off as its
// windup completes.
func weaponSwings(lines []timelineLine) map[string][]float64 {
	raptor := core.ActionID{SpellID: raptorStrikeRank.ID}.String()
	swings := map[string][]float64{}
	for _, l := range lines {
		switch l.text {
		case "Completed cast " + mhID, "Completed cast " + raptor:
			swings[mhID] = append(swings[mhID], l.at)
		case "Completed cast " + ohID:
			swings[ohID] = append(swings[ohID], l.at)
		case "Completed cast " + shotID:
			swings[shotID] = append(swings[shotID], l.at)
		}
	}
	return swings
}

func near(a, b float64) bool { return math.Abs(a-b) <= logStampSlack }

// Every swing starts its weapon's timer, logged under the weapon's own id, and the timers agree with the fight: a weapon
// never swings before its timer is ready, and swings when it is ready unless a logged wait holds it until then. A speed
// change that moves a running timer logs its new ready time.
func TestSwingTimerLogLines(t *testing.T) {
	moved := 0
	for _, fight := range []struct {
		name          string
		melee, ranged bool
	}{{"dw_auto_weave", true, true}, {"2h_auto_weave", true, true}, {"dw_pure_melee", true, false}, {"bm_ranged", false, true}} {
		name := fight.name
		lines := timelineBuildFight(t, name)
		swings := weaponSwings(lines)
		waits := swingWaits(lines)
		starts := map[string]int{}
		// the last ready time logged for each weapon, and since when no swing of it came
		ready := map[string]float64{}
		lastSwing := map[string]float64{}
		swingIdx := map[string]int{}
		for _, l := range lines {
			m := timerRe.FindStringSubmatch(l.text)
			if m == nil {
				continue
			}
			weapon, readyAt := m[1], atof(m[3])
			if weapon != mhID && weapon != ohID && weapon != shotID {
				t.Errorf("%s: a swing timer under %s; want the weapon's own id", name, weapon)
				continue
			}
			if m[2] == "" {
				moved++
				ready[weapon] = readyAt
				continue
			}
			starts[weapon]++
			i := swingIdx[weapon]
			if i >= len(swings[weapon]) || !near(swings[weapon][i], l.at) {
				t.Errorf("%s: %s timer started at %.2f s with no swing of it then", name, weapon, l.at)
				continue
			}
			if i > 0 {
				s, r := swings[weapon][i], ready[weapon]
				if s < r-logStampSlack {
					t.Errorf("%s: %s swung at %.2f s, before its timer was ready at %.3f s", name, weapon, s, r)
				}
				if s > r+logStampSlack {
					held := false
					for _, w := range waits {
						held = held || (w.weapon == weapon && w.at >= lastSwing[weapon] && near(w.to, s))
					}
					if !held {
						t.Errorf("%s: %s swung at %.2f s, its timer ready at %.3f s, and no logged wait holds it until then", name, weapon, s, r)
					}
				}
			}
			swingIdx[weapon] = i + 1
			lastSwing[weapon] = l.at
			ready[weapon] = readyAt
		}
		for weapon, s := range swings {
			if starts[weapon] != len(s) {
				t.Errorf("%s: %d %s swings, %d timer starts", name, len(s), weapon, starts[weapon])
			}
		}
		if fight.melee && len(swings[mhID]) < 10 || fight.ranged && len(swings[shotID]) < 10 {
			t.Errorf("%s: %d main-hand swings and %d Auto Shots; want the fight to play", name, len(swings[mhID]), len(swings[shotID]))
		}
	}
	if moved == 0 {
		t.Error("no running timer moved by a speed change in four fights; want some")
	}
}

// The waits tell the truth: a main hand or off hand that came ready out of melee range waits from its timer's ready to
// the arrival, then the swing retry runs in half-second steps to the swing; an Auto Shot ready during a walk waits for it
// to end, then for the 0.6 s from the stop to the shot.
func TestSwingWaitLogLines(t *testing.T) {
	counts := map[string]int{}
	for _, name := range []string{"dw_auto_weave", "2h_auto_weave"} {
		lines := timelineBuildFight(t, name)
		swings := weaponSwings(lines)
		walkEnds := map[float64]bool{}
		for _, l := range lines {
			if l.text == "Aura faded: "+walkID {
				walkEnds[l.at] = true
			}
		}
		nearAny := func(times []float64, at float64) bool {
			for _, s := range times {
				if near(s, at) {
					return true
				}
			}
			return false
		}
		swungAt := func(weapon string, at float64) bool { return nearAny(swings[weapon], at) }
		arrivedAt := func(at float64) bool {
			for e := range walkEnds {
				if near(e, at) {
					return true
				}
			}
			return false
		}
		ready := map[string]float64{}
		for _, l := range lines {
			if m := timerRe.FindStringSubmatch(l.text); m != nil {
				ready[m[1]] = atof(m[3])
			}
			m := waitRe.FindStringSubmatch(l.text)
			if m == nil {
				continue
			}
			w := swingWaits([]timelineLine{l})[0]
			counts[w.reason]++
			switch w.reason {
			case "range":
				if math.Abs(w.from-math.Max(ready[w.weapon], 0)) > 0.0015 {
					t.Errorf("%s: %s out of range from %.3f s; want from its timer's ready at %.3f s", name, w.weapon, w.from, ready[w.weapon])
				}
				if !walkEnds[w.at] {
					t.Errorf("%s: %s out of range until %.3f s, with no walk ending then", name, w.weapon, w.to)
				}
			case "retry":
				if !swungAt(w.weapon, w.to) {
					t.Errorf("%s: a %s retry ends at %.3f s, but it did not swing then", name, w.weapon, w.to)
				}
				if steps := (w.to - w.origin) / 0.5; math.Abs(steps-math.Round(steps)) > 0.004 {
					t.Errorf("%s: a %s retry swings %.3f s after %.3f s; want whole half-second steps", name, w.weapon, w.to-w.origin, w.origin)
				}
				// from the arrival, or from the timer's ready when it ran out after the arrival
				if !arrivedAt(w.from) && math.Abs(w.from-ready[w.weapon]) > 0.0015 {
					t.Errorf("%s: a %s retry from %.3f s; want from the arrival or its timer's ready at %.3f s", name, w.weapon, w.from, ready[w.weapon])
				}
			case "windup":
				if math.Abs(w.to-w.stoppedAt-core.ForeverAutoShotStandStill.Seconds()) > 0.0015 {
					t.Errorf("%s: a windup from the stop at %.3f s to %.3f s; want %.1f s", name, w.stoppedAt, w.to, core.ForeverAutoShotStandStill.Seconds())
				}
				if !walkEnds[w.at] {
					t.Errorf("%s: a windup stopped at %.3f s with no walk ending then", name, w.stoppedAt)
				}
				// a windup the fight's end cuts short never fires
				if w.from < w.to-0.001 && w.to <= lines[len(lines)-1].at && !swungAt(shotID, w.to) {
					t.Errorf("%s: a windup holds the Auto Shot to %.3f s, but it did not fire then", name, w.to)
				}
			case "moving":
				if !walkEnds[w.at] || w.from > w.to {
					t.Errorf("%s: an Auto Shot held while moving from %.3f s to %.3f s; want it to the walk's end", name, w.from, w.to)
				}
			}
		}
	}
	for _, reason := range []string{"range", "retry", "windup", "moving"} {
		if counts[reason] == 0 {
			t.Errorf("no %s wait in two weave fights; want some", reason)
		}
	}
}

// A walk-in and walk-out by hand: in melee from 2 s to 6 s the Auto Shot comes ready (too close to shoot) and waits
// until the walk out starts, through the walk, and for the windup from the stop. Walking out right on arrival leaves
// before the main hand's swing retry goes off.
func TestSwingWaitCloseAndLeft(t *testing.T) {
	at := func(v string) string { return `{"const":{"val":"` + v + `"}}` }
	cmp := func(op, lhs, rhs string) string {
		return `{"cmp":{"op":"` + op + `","lhs":` + lhs + `,"rhs":` + rhs + `}}`
	}
	distance := `{"unitDistance":{"sourceUnit":{"type":"Self"}}}`
	now := `{"currentTime":{}}`
	move := func(yd string) string { return `"move":{"rangeFromTarget":` + at(yd) + `}` }
	when := func(cond, action string) string {
		return `{"action":{"condition":{"and":{"vals":[` + cond + `]}},` + action + `}}`
	}
	fight := func(stayUntil string) []timelineLine {
		rot := resultsRotation(t, `{"type":"TypeAPL","priorityList":[`+
			when(cmp("OpGe", now, at("1s"))+`,`+cmp("OpLt", now, at("1.5s"))+`,`+cmp("OpGt", distance, at("5")), move("5"))+`,`+
			when(cmp("OpGe", now, at(stayUntil))+`,`+cmp("OpLe", distance, at("5")), move("12"))+`]}`)
		player := &proto.Player{
			Name:               "Walker",
			Race:               proto.Race_RaceOrc,
			Class:              proto.Class_ClassHunter,
			Equipment:          core.GetGearSet("../../ui/specs/hunter/dps/gear_sets", "p1").GearSet,
			TalentsString:      DefaultMMTalents,
			Spec:               DefaultOptions,
			Rotation:           rot,
			DistanceFromTarget: 12,
		}
		encounter := core.MakeSingleTargetEncounter(0)
		encounter.Duration = 12
		return timelineFight(t, player, &proto.IndividualSimSettings{PartyBuffs: &proto.PartyBuffs{}, RaidBuffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}}, encounter)
	}
	walks := func(lines []timelineLine) []float64 {
		var starts []float64
		for _, l := range lines {
			if strings.HasPrefix(l.text, "Casting "+walkID+" (") && l.at > 0 {
				starts = append(starts, l.at)
			}
		}
		return starts
	}

	lines := fight("6s")
	waits := swingWaits(lines)
	shots := weaponSwings(lines)[shotID]
	starts := walks(lines)
	if len(starts) != 2 || len(shots) < 2 {
		t.Fatalf("staying in melee: walks at %v and Auto Shots at %v; want a walk in, a walk out and shots", starts, shots)
	}
	var close, moving, windup *swingWait
	for i := range waits {
		switch w := &waits[i]; w.reason {
		case "close":
			close = w
		case "moving":
			moving = w
		case "windup":
			windup = w
		}
	}
	if close == nil || moving == nil || windup == nil {
		t.Fatalf("staying in melee: waits %+v; want close, moving and windup", waits)
	}
	t.Logf("staying in melee: walks at %v, Auto Shots at %v, waits %+v", starts, shots, waits)
	firstReady := 0.0
	for _, l := range lines {
		if m := timerRe.FindStringSubmatch(l.text); m != nil && m[1] == shotID {
			firstReady = atof(m[3])
			break
		}
	}
	if math.Abs(close.from-firstReady) > 0.0015 || !near(close.to, starts[1]) {
		t.Errorf("too close from %.3f s to %.3f s; want from the Auto Shot's ready at %.3f s to the walk out at %.2f s", close.from, close.to, firstReady, starts[1])
	}
	if math.Abs(moving.from-close.to) > 0.0015 || math.Abs(moving.to-windup.stoppedAt) > 0.0015 {
		t.Errorf("moving from %.3f s to %.3f s; want from the walk out at %.3f s to the stop at %.3f s", moving.from, moving.to, close.to, windup.stoppedAt)
	}
	if !near(shots[1], windup.to) {
		t.Errorf("the second Auto Shot at %.2f s; want it as the windup ends at %.3f s", shots[1], windup.to)
	}

	lines = fight("1.5s")
	waits = swingWaits(lines)
	mh := weaponSwings(lines)[mhID]
	starts = walks(lines)
	var left *swingWait
	for i := range waits {
		if waits[i].reason == "left" && waits[i].weapon == mhID {
			left = &waits[i]
		}
	}
	if left == nil || len(starts) < 2 {
		t.Fatalf("walking out on arrival: waits %+v, walks at %v; want the main hand's retry left", waits, starts)
	}
	t.Logf("walking out on arrival: walks at %v, main-hand swings at %v, waits %+v", starts, mh, waits)
	if !near(left.to, starts[1]) {
		t.Errorf("left at %.3f s; want as the walk out starts at %.2f s", left.to, starts[1])
	}
	for _, s := range mh {
		if s > left.from-logStampSlack && s < left.to+logStampSlack {
			t.Errorf("the main hand swung at %.2f s, during the retry it left at %.3f s", s, left.to)
		}
	}
}

// A GCD free line names what could have been pressed while the GCD sat unused: it is written in a rotation pass that
// leaves the GCD free (the pass holds it 50 ms, "Extending GCD"), only when the list changes, so the same list comes
// again only after a pass with the GCD busy; and it names only spells the hunter casts on the GCD.
func TestGcdFreeLogLines(t *testing.T) {
	gcdCastRe := regexp.MustCompile(`^Casting (\{[^}]*\}) \(.*GCD = 1\.5s`)
	for _, name := range []string{"dw_auto_weave", "2h_auto_weave"} {
		lines := timelineBuildFight(t, name)
		gcdSpells := map[string]bool{}
		for _, l := range lines {
			if m := gcdCastRe.FindStringSubmatch(l.text); m != nil {
				gcdSpells[m[1]] = true
			}
		}
		count, passFree, busySince, freeLinePending := 0, false, true, false
		last := ""
		for _, l := range lines {
			switch {
			case strings.HasPrefix(l.text, "Extending GCD for "):
				passFree = true
			case strings.HasPrefix(l.text, "Pausing rotation for "):
				if freeLinePending && !passFree {
					t.Errorf("%s: a GCD free line at %.2f s in a pass that left the GCD busy", name, l.at)
				}
				if !passFree {
					busySince = true
				}
				passFree, freeLinePending = false, false
			case strings.HasPrefix(l.text, "GCD free, "):
				count++
				freeLinePending = true
				if l.text == last && !busySince {
					t.Errorf("%s: %q again at %.2f s with no pass with the GCD busy since", name, l.text, l.at)
				}
				for _, id := range regexp.MustCompile(`\{[^}]*\}`).FindAllString(l.text, -1) {
					if !gcdSpells[id] {
						t.Errorf("%s: a free GCD at %.2f s could press %s, which the hunter never casts on the GCD", name, l.at, id)
					}
				}
				last, busySince = l.text, false
			}
		}
		if count == 0 {
			t.Errorf("%s: no GCD free line; want some", name)
		}
	}
}

// The off hand's half-swing wait from the moment the melee attack turns on is a log-only cast the timeline draws: it
// runs half the off hand's swing time, ends with its own completion, and no off-hand swing lands inside it.
func TestOffHandWaitMarkers(t *testing.T) {
	waitID := core.OffHandWaitID.String()
	castRe := regexp.MustCompile(`^Casting ` + regexp.QuoteMeta(waitID) + ` \(Cost = 0\.000, Cast Time = ([0-9.]+)(m?s), GCD = 0s`)
	lines := timelineBuildFight(t, "dw_auto_weave")
	ohSwings := weaponSwings(lines)[ohID]
	// the off hand's swing time from its timer started nearest in the fight (attack speed changes)
	type ohTimer struct{ at, swingTime float64 }
	var ohTimers []ohTimer
	var ends []float64
	for _, l := range lines {
		if m := timerRe.FindStringSubmatch(l.text); m != nil && m[1] == ohID && m[2] != "" {
			ohTimers = append(ohTimers, ohTimer{l.at, atof(m[3]) - l.at})
		}
		if l.text == "Completed cast "+waitID {
			ends = append(ends, l.at)
		}
	}
	if len(ohTimers) == 0 {
		t.Fatal("no off-hand timer in a DW weave fight")
	}
	markers := 0
	for _, l := range lines {
		m := castRe.FindStringSubmatch(l.text)
		if m == nil || l.at < 0 {
			continue
		}
		markers++
		wait := atof(m[1])
		if m[2] == "ms" {
			wait /= 1000
		}
		// the swing time read off the off hand's timers on either side of the wait (one may follow a speed change); the
		// stamps' 0.01 s leave it a few ms out
		half := false
		var swingTimes []float64
		for i, o := range ohTimers {
			if o.at <= l.at && (i+1 == len(ohTimers) || ohTimers[i+1].at > l.at) || o.at > l.at && (i == 0 || ohTimers[i-1].at <= l.at) {
				swingTimes = append(swingTimes, o.swingTime)
				half = half || math.Abs(wait-o.swingTime/2) <= 0.004
			}
		}
		if !half {
			t.Errorf("an off-hand wait of %.3f s at %.2f s; want half the off hand's swing time (%v)", wait, l.at, swingTimes)
		}
		ended := l.at+wait > lines[len(lines)-1].at // still running as the fight ends
		for _, e := range ends {
			ended = ended || near(e, l.at+wait)
		}
		if !ended {
			t.Errorf("the off-hand wait from %.2f s does not complete %.3f s later", l.at, wait)
		}
		for _, s := range ohSwings {
			if s > l.at+logStampSlack && s < l.at+wait-logStampSlack {
				t.Errorf("the off hand swung at %.2f s inside its wait from %.2f s to %.2f s", s, l.at, l.at+wait)
			}
		}
	}
	if markers < 10 {
		t.Errorf("%d off-hand waits in a DW weave fight; want one each walk in", markers)
	}
}

// The move reaction in front of each walk is a log-only cast the timeline draws: 0.1 s from the decision to move to the
// walk's start, which a walk decided on the way starts at the end of the walk under way instead.
func TestMoveReactionMarkers(t *testing.T) {
	reactionID := core.MoveReactionID.String()
	lines := timelineBuildFight(t, "dw_auto_weave")
	var reactions, reactionEnds, walkStarts []float64
	walkEnds := map[float64]bool{}
	decided := map[float64]bool{}
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l.text, "Casting "+reactionID+" (Cost = 0.000, Cast Time = 100ms, GCD = 0s"):
			reactions = append(reactions, l.at)
		case l.text == "Completed cast "+reactionID:
			reactionEnds = append(reactionEnds, l.at)
		case strings.HasPrefix(l.text, "Casting "+walkID+" (") && l.at > 0:
			walkStarts = append(walkStarts, l.at)
		case l.text == "Aura faded: "+walkID:
			walkEnds[l.at] = true
		case strings.HasPrefix(l.text, "[DEBUG] Moving to "):
			decided[l.at] = true
		}
	}
	// the last one may still run as the fight ends
	if len(reactions) < 10 || len(reactionEnds) < len(reactions)-1 || len(reactionEnds) > len(reactions) {
		t.Fatalf("%d move reactions and %d ends; want one end each, a reaction each walk", len(reactions), len(reactionEnds))
	}
	for i, end := range reactionEnds {
		r := reactions[i]
		if !decided[r] {
			t.Errorf("a move reaction at %.2f s with no decision to move then", r)
		}
		if math.Abs(end-r-0.1) > 0.011 {
			t.Errorf("the move reaction from %.2f s ends at %.2f s; want 0.1 s later", r, end)
		}
	}
	for _, w := range walkStarts {
		after := walkEnds[w]
		for _, e := range reactionEnds {
			after = after || e == w
		}
		if !after {
			t.Errorf("a walk at %.2f s after no move reaction and no walk ending then", w)
		}
	}
}

// Every Auto Shot logs its cast start before it completes, the pull's first one too (it wound up before the pull), or
// the timeline pairs each shot with the one before it.
func TestEveryAutoShotLogsItsCastStart(t *testing.T) {
	autoShot := core.ActionID{OtherID: proto.OtherAction_OtherActionShoot}.String()
	for _, name := range []string{"dw_auto_weave", "2h_auto_weave"} {
		started, shots := false, 0
		for _, l := range timelineBuildFight(t, name) {
			switch {
			case strings.HasPrefix(l.text, "Casting "+autoShot+" ("):
				started = true
			case l.text == "Completed cast "+autoShot:
				if !started {
					t.Errorf("%s: an Auto Shot completes at %.2f s with no cast start logged since the last one", name, l.at)
				}
				started = false
				shots++
			}
		}
		if shots < 10 {
			t.Errorf("%s: %d Auto Shots logged; want a fight's worth", name, shots)
		}
	}
}

// Logging changes nothing: the same fights with every iteration logged deal the same damage to the last digit (the
// log's markers add events of their own).
func TestTimelineLogsChangeNothing(t *testing.T) {
	for _, name := range []string{"dw_auto_weave", "2h_auto_weave"} {
		build := timelineBuild(t, name)
		dps := func(debug bool) float64 {
			res := core.RunRaidSim(&proto.RaidSimRequest{
				Raid:       core.SinglePlayerRaidProto(build.Player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
				Encounter:  build.Encounter,
				SimOptions: &proto.SimOptions{Iterations: 20, RandomSeed: 7, Debug: debug},
			})
			if res.Error != nil {
				t.Fatalf("sim error: %s", res.Error.Message)
			}
			return res.RaidMetrics.Parties[0].Players[0].Dps.Avg
		}
		if quiet, logged := dps(false), dps(true); quiet != logged {
			t.Errorf("%s: %.6f DPS unlogged, %.6f logged; want the same", name, quiet, logged)
		}
	}
}
