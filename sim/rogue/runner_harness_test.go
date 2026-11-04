package rogue

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

// Runs a RaidSimRequest (ARC_REQ, JSON) and prints the rogue's DPS and a per-ability table, for the sims runner.
// ARC_ITERS overrides the iterations. ARC_EP=<stat names> prints stat weights instead, against attack power as the
// rogue page weighs them.
func TestRunnerRequest(t *testing.T) {
	path := os.Getenv("ARC_REQ")
	if path == "" {
		t.Skip("ARC_REQ not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	req := &proto.RaidSimRequest{}
	if err := protojson.Unmarshal(raw, req); err != nil {
		t.Fatalf("request: %v", err)
	}
	if v := os.Getenv("ARC_ITERS"); v != "" {
		n, _ := strconv.Atoi(v)
		req.SimOptions.Iterations = int32(n)
	}
	if names := os.Getenv("ARC_EP"); names != "" {
		printRogueStatWeights(t, req, strings.Split(names, ","))
		return
	}
	res := core.RunRaidSim(req)
	if res.Error != nil {
		t.Fatalf("sim error: %s", res.Error.Message)
	}
	iters := float64(req.SimOptions.Iterations)
	player := res.RaidMetrics.Parties[0].Players[0]
	fmt.Printf("DPS %.1f (stdev %.1f)  raid DPS incl. pets %.1f\n", player.Dps.Avg, player.Dps.Stdev, res.RaidMetrics.Dps.Avg)
	type row struct {
		name           string
		casts, damage  float64
		hits, crits    float64
		misses, dodges float64
	}
	var rows []row
	for _, a := range player.Actions {
		r := row{name: fmt.Sprintf("%d/%d/%d", a.Id.GetSpellId(), a.Id.GetOtherId(), a.Id.Tag)}
		for _, tm := range a.Targets {
			r.casts += float64(tm.Casts)
			r.damage += tm.Damage
			r.hits += float64(tm.Hits)
			r.crits += float64(tm.Crits)
			r.misses += float64(tm.Misses)
			r.dodges += float64(tm.Dodges)
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].damage > rows[j].damage })
	for _, r := range rows {
		fmt.Printf("  %-22s casts %7.2f  damage %9.1f  hits %7.2f  crits %6.2f  misses %5.2f  dodges %5.2f\n",
			r.name, r.casts/iters, r.damage/iters, r.hits/iters, r.crits/iters, r.misses/iters, r.dodges/iters)
	}
	// Uptimes on their own prefix, so the ability table's fingerprint stays what it was.
	fmt.Printf("AURA fight %.2f\n", res.AvgIterationDuration)
	for _, a := range player.Auras {
		fmt.Printf("AURA player %d/%d/%d uptime %.2f procs %.2f\n", a.Id.GetSpellId(), a.Id.GetOtherId(), a.Id.Tag, a.UptimeSecondsAvg, a.ProcsAvg)
	}
	for _, target := range res.EncounterMetrics.Targets {
		for _, a := range target.Auras {
			fmt.Printf("AURA target %d/%d/%d uptime %.2f procs %.2f\n", a.Id.GetSpellId(), a.Id.GetOtherId(), a.Id.Tag, a.UptimeSecondsAvg, a.ProcsAvg)
		}
	}
}

func printRogueStatWeights(t *testing.T, req *proto.RaidSimRequest, names []string) {
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
		EpReferenceStat:    proto.Stat_StatAttackPower,
		HpsReferenceStat:   proto.Stat_StatAttackPower,
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
