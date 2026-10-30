package hunter

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// The seed and iterations behind the site's reference numbers.
const (
	arcRefSeed       = 20261012
	arcRefIterations = 20000
)

// The request the site's reference numbers come from for a build file (IndividualSimSettings, as the site exports it).
func arcBuildRequest(raw []byte) (*proto.RaidSimRequest, error) {
	build := &proto.IndividualSimSettings{}
	if err := protojson.Unmarshal(raw, build); err != nil {
		return nil, err
	}
	return &proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(build.Player, build.PartyBuffs, build.RaidBuffs, build.Debuffs),
		Encounter:  build.Encounter,
		SimOptions: &proto.SimOptions{Iterations: arcRefIterations, RandomSeed: arcRefSeed},
	}, nil
}

// A build file runs as the site's reference numbers do: its player, buffs, debuffs and encounter, the reference seed,
// 20000 iterations.
func TestArcBuildRequestIsTheReferenceRun(t *testing.T) {
	raw, err := os.ReadFile("../../ui/specs/hunter/dps/builds/dw_auto_weave.build.json")
	if err != nil {
		t.Fatal(err)
	}
	build := &proto.IndividualSimSettings{}
	if err := protojson.Unmarshal(raw, build); err != nil {
		t.Fatal(err)
	}
	req, err := arcBuildRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if req.SimOptions.RandomSeed != 20261012 || req.SimOptions.Iterations != 20000 {
		t.Errorf("seed %d, %d iterations; want 20261012 and 20000", req.SimOptions.RandomSeed, req.SimOptions.Iterations)
	}
	party := req.Raid.Parties[0]
	if len(party.Players) != 1 || party.Players[0].Name != build.Player.Name || party.Players[0].Equipment.String() != build.Player.Equipment.String() {
		t.Error("the raid is not the build's player alone")
	}
	if party.Buffs.String() != build.PartyBuffs.String() || req.Raid.Buffs.String() != build.RaidBuffs.String() || req.Raid.Debuffs.String() != build.Debuffs.String() || req.Encounter.String() != build.Encounter.String() {
		t.Error("the build's buffs, debuffs or encounter did not carry over")
	}
}

// Runs a build file (ARC_BUILD) or a converted build (ARC_REQ, a RaidSimRequest as JSON) and prints DPS and a
// per-ability table. ARC_ITERS overrides the iterations, ARC_DEBUG=1 prints the log of one iteration, ARC_EP=<stat
// names> prints stat weights.
func TestArcSpikeSettings(t *testing.T) {
	path, fromBuild := os.Getenv("ARC_REQ"), os.Getenv("ARC_BUILD") != ""
	if fromBuild {
		path = os.Getenv("ARC_BUILD")
	}
	if path == "" {
		t.Skip("ARC_BUILD or ARC_REQ not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	req := &proto.RaidSimRequest{}
	if fromBuild {
		req, err = arcBuildRequest(raw)
	} else {
		err = protojson.Unmarshal(raw, req)
	}
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if v := os.Getenv("ARC_ITERS"); v != "" {
		n, _ := strconv.Atoi(v)
		req.SimOptions.Iterations = int32(n)
	}
	if names := os.Getenv("ARC_EP"); names != "" {
		printStatWeights(t, req, strings.Split(names, ","))
		return
	}
	if os.Getenv("ARC_STATS") != "" {
		cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: req.Raid, Encounter: req.Encounter})
		ps := cs.RaidStats.Parties[0].Players[0]
		for _, part := range []struct {
			name string
			us   *proto.UnitStats
		}{{"base", ps.BaseStats}, {"gear", ps.GearStats}, {"talents", ps.TalentsStats}, {"buffs", ps.BuffsStats}, {"consumes", ps.ConsumesStats}, {"final", ps.FinalStats}} {
			if os.Getenv("ARC_STATS") == "short" {
				st := part.us.Stats
				fmt.Printf("%-9s Str %6.1f Agi %6.1f Int %6.1f AP %7.1f RAP %7.1f\n", part.name, st[proto.Stat_StatStrength], st[proto.Stat_StatAgility], st[proto.Stat_StatIntellect], st[proto.Stat_StatAttackPower], st[proto.Stat_StatRangedAttackPower])
				continue
			}
			fmt.Printf("%-6s", part.name)
			for i, v := range part.us.Stats {
				if v != 0 {
					fmt.Printf(" %s %.1f", proto.Stat(i).String(), v)
				}
			}
			for i, v := range part.us.PseudoStats {
				if v != 0 {
					fmt.Printf(" %s %.2f", proto.PseudoStat(i).String(), v)
				}
			}
			fmt.Println()
		}
		return
	}
	if os.Getenv("ARC_DEBUG") != "" {
		req.SimOptions.Debug = true
		req.SimOptions.Iterations = 1
	}
	res := core.RunRaidSim(req)
	if res.Error != nil {
		t.Fatalf("sim error: %s", res.Error.Message)
	}
	if res.Logs != "" {
		fmt.Println(res.Logs)
	}
	iters := float64(req.SimOptions.Iterations)
	player := res.RaidMetrics.Parties[0].Players[0]
	fmt.Printf("DPS %.1f (stdev %.1f)  raid DPS incl. pets %.1f\n", player.Dps.Avg, player.Dps.Stdev, res.RaidMetrics.Dps.Avg)
	type row struct {
		name                                        string
		casts, hits, crits, misses, dodges, glances float64
		ticks, damage                               float64
	}
	var rows []row
	add := func(owner string, actions []*proto.ActionMetrics) {
		for _, a := range actions {
			r := row{name: owner + actionName(a.Id)}
			for _, tm := range a.Targets {
				r.casts += float64(tm.Casts)
				r.hits += float64(tm.Hits)
				r.crits += float64(tm.Crits)
				r.misses += float64(tm.Misses)
				r.dodges += float64(tm.Dodges)
				r.glances += float64(tm.Glances)
				r.ticks += float64(tm.Ticks)
				r.damage += tm.Damage
			}
			if r.damage > 0 || r.casts > 0 {
				rows = append(rows, r)
			}
		}
	}
	add("", player.Actions)
	for _, pet := range player.Pets {
		add(pet.Name+" ", pet.Actions)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].damage > rows[j].damage })
	for _, r := range rows {
		fmt.Printf("  %-34s casts %6.1f hits %6.1f crit %5.1f miss %4.1f dodge %4.1f glance %5.1f ticks %5.1f dmg/fight %8.0f\n",
			r.name, r.casts/iters, r.hits/iters, r.crits/iters, r.misses/iters, r.dodges/iters, r.glances/iters, r.ticks/iters, r.damage/iters)
	}
	// mana gained (+) and spent (-) per source, per fight
	if os.Getenv("ARC_MANA") != "" {
		var gained, spent float64
		for _, r := range player.Resources {
			if r.Type != proto.ResourceType_ResourceTypeMana {
				continue
			}
			g := r.ActualGain / iters
			fmt.Printf("  mana %-28s %6.0f\n", actionName(r.Id), g)
			if g > 0 {
				gained += g
			} else {
				spent -= g
			}
		}
		fmt.Printf("  mana gained %.0f, spent %.0f per fight\n", gained, spent)
	}
}

// The weights the page's Calculate button gives for the named stats (proto Stat or PseudoStat names), against Agility as
// the hunter page weighs them, at the request's iterations and seed: one "EP <name> <weight> <stdev>" line each.
func printStatWeights(t *testing.T, req *proto.RaidSimRequest, names []string) {
	var statsToWeigh []proto.Stat
	var pseudoToWeigh []proto.PseudoStat
	for _, name := range names {
		if v, ok := proto.Stat_value[name]; ok {
			statsToWeigh = append(statsToWeigh, proto.Stat(v))
		} else if v, ok := proto.PseudoStat_value[name]; ok {
			pseudoToWeigh = append(pseudoToWeigh, proto.PseudoStat(v))
		} else {
			t.Fatalf("unknown stat %s", name)
		}
	}
	party := req.Raid.Parties[0]
	res := core.StatWeights(&proto.StatWeightsRequest{
		Player:             party.Players[0],
		RaidBuffs:          req.Raid.Buffs,
		PartyBuffs:         party.Buffs,
		Debuffs:            req.Raid.Debuffs,
		Encounter:          req.Encounter,
		SimOptions:         req.SimOptions,
		StatsToWeigh:       statsToWeigh,
		PseudoStatsToWeigh: pseudoToWeigh,
		EpReferenceStat:    proto.Stat_StatAgility,
		HpsReferenceStat:   proto.Stat_StatAgility,
	})
	if res.Error != nil {
		t.Fatalf("stat weights error: %s", res.Error.Message)
	}
	ep, sd := res.Dps.EpValues, res.Dps.EpValuesStdev
	for _, s := range statsToWeigh {
		fmt.Printf("EP %s %.4f %.4f\n", s, ep.Stats[s], sd.Stats[s])
	}
	for _, s := range pseudoToWeigh {
		fmt.Printf("EP %s %.4f %.4f\n", s, ep.PseudoStats[s], sd.PseudoStats[s])
	}
}

func actionName(id *proto.ActionID) string {
	s := ""
	switch {
	case id.GetSpellId() != 0:
		s = fmt.Sprintf("spell %d", id.GetSpellId())
	case id.GetItemId() != 0:
		s = fmt.Sprintf("item %d", id.GetItemId())
	default:
		s = fmt.Sprintf("other %d", id.GetOtherId())
	}
	if id.GetTag() != 0 {
		s += fmt.Sprintf(" t%d", id.GetTag())
	}
	return s
}
