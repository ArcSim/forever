package hunter

import (
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
	"google.golang.org/protobuf/encoding/protojson"
	googleProto "google.golang.org/protobuf/proto"
)

func resultsRotation(t *testing.T, aplJSON string) *proto.APLRotation {
	t.Helper()
	rot := &proto.APLRotation{}
	if err := protojson.Unmarshal([]byte(aplJSON), rot); err != nil {
		t.Fatalf("rotation: %v", err)
	}
	return rot
}

func runResultsSim(t *testing.T, raid *proto.Raid, encounter *proto.Encounter, iterations int32) *proto.UnitMetrics {
	t.Helper()
	res := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 20261026},
	})
	if res.Error != nil {
		t.Fatalf("sim error: %s", res.Error.Message)
	}
	return res.RaidMetrics.Parties[0].Players[0]
}

// The APM counts what the player does over a 60 s fight: each Arcane Shot, the walk in at 10 s, the Start Attack
// once there, and the walk back out at 20 s. The Auto Shots, the melee swings and the pre-pull Hunter's Mark count
// nothing.
func TestAPMCountsPresses(t *testing.T) {
	at := func(v string) string { return `{"const":{"val":"` + v + `"}}` }
	cmp := func(op, lhs, rhs string) string {
		return `{"cmp":{"op":"` + op + `","lhs":` + lhs + `,"rhs":` + rhs + `}}`
	}
	distance := `{"unitDistance":{"sourceUnit":{"type":"Self"}}}`
	// the action, once, in the second from s while the distance compares (op) to yd
	once := func(s int, op string, yd int, action string) string {
		return `{"action":{"condition":{"and":{"vals":[` +
			cmp("OpGe", `{"currentTime":{}}`, at(strconv.Itoa(s)+"s")) + `,` +
			cmp("OpLt", `{"currentTime":{}}`, at(strconv.Itoa(s+1)+"s")) + `,` +
			cmp(op, distance, at(strconv.Itoa(yd))) + `]}},` + action + `}}`
	}
	move := func(yd int) string { return `"move":{"rangeFromTarget":` + at(strconv.Itoa(yd)) + `}` }
	mark := `{"castSpell":{"spellId":{"spellId":` + strconv.Itoa(int(huntersMarkRank.ID)) + `}}}`
	rot := resultsRotation(t, `{"type":"TypeAPL","manualStartAttack":true,`+
		`"prepullActions":[{"action":`+mark+`,"doAtValue":`+at("-2s")+`}],"priorityList":[`+
		once(10, "OpGt", 5, move(5))+`,`+
		once(12, "OpLe", 5, `"startAttack":{}`)+`,`+
		once(20, "OpLt", 12, move(12))+`,`+
		`{"action":{"castSpell":{"spellId":{"spellId":`+strconv.Itoa(int(arcaneShotRank.ID))+`}}}}]}`)

	player := &proto.Player{
		Name:               "Presser",
		Race:               proto.Race_RaceOrc,
		Class:              proto.Class_ClassHunter,
		Equipment:          core.GetGearSet("../../ui/specs/hunter/dps/gear_sets", "p1").GearSet,
		TalentsString:      DefaultMMTalents,
		Spec:               DefaultOptions,
		Rotation:           rot,
		DistanceFromTarget: 12,
	}
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Duration = 60
	um := runResultsSim(t, core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}), encounter, 1)

	var shots, autos, swings, marks int32
	for _, a := range um.Actions {
		for _, tg := range a.Targets {
			switch {
			case a.Id.GetSpellId() == arcaneShotRank.ID:
				shots += tg.Casts
			case a.Id.GetSpellId() == huntersMarkRank.ID:
				marks += tg.Casts
			case a.Id.GetOtherId() == proto.OtherAction_OtherActionShoot:
				autos += tg.Casts
			case a.Id.GetOtherId() == proto.OtherAction_OtherActionAttack:
				swings += tg.Casts
			}
		}
	}
	if shots == 0 || autos == 0 || swings == 0 || marks != 1 {
		t.Fatalf("%d Arcane Shots, %d Auto Shots, %d swings and %d Hunter's Marks: the fight did not play", shots, autos, swings, marks)
	}
	t.Logf("APM %.2f: %d Arcane Shots, %d Auto Shots, %d swings", um.Apm, shots, autos, swings)
	if want := float64(shots + 3); math.Abs(um.Apm-want) > 1e-9 {
		t.Errorf("APM %.2f, want %.0f (%d Arcane Shots, 2 walks and a Start Attack in a 60 s fight; %d Auto Shots and %d swings not counted)",
			um.Apm, want, shots, autos, swings)
	}
}

// Expose Prey is its own buff, and each proc ends one way: a Mongoose Bite spends it, it runs out unused (one still up
// when the fight ends counts as neither), or a new proc refreshes it while it is up. A marker counts each: used
// matches the bites, as nothing attacks the hunter to open a bite window of its own, and the procs that were not
// refreshes, less the used and expired ones, leave at most one a fight. Shooting from range, nothing is used and the
// fights that end with it up leave some unaccounted.
func TestExposePreyMarkers(t *testing.T) {
	const iterations = 50
	raw, err := os.ReadFile("../../ui/specs/hunter/dps/builds/2h_auto_weave.build.json")
	if err != nil {
		t.Fatal(err)
	}
	type counts struct{ gained, used, expired, refreshed, bites float64 }
	run := func(build *proto.IndividualSimSettings) counts {
		um := runResultsSim(t, core.SinglePlayerRaidProto(build.Player, build.PartyBuffs, build.RaidBuffs, build.Debuffs), build.Encounter, iterations)
		procs := map[int32]float64{}
		for _, a := range um.Auras {
			if a.Id.GetSpellId() == exposePreyWindow.ID {
				procs[a.Id.Tag] = a.ProcsAvg
			}
		}
		c := counts{gained: procs[0], used: procs[1], expired: procs[2], refreshed: procs[3]}
		for _, a := range um.Actions {
			if a.Id.GetSpellId() == mongooseBiteRank.ID {
				for _, tm := range a.Targets {
					c.bites += float64(tm.Casts) / iterations // a bite that misses spends it too
				}
			}
		}
		t.Logf("Expose Prey a fight: %.2f procs, %.2f used, %.2f expired unused, %.2f refreshed, %.2f Mongoose Bites",
			c.gained, c.used, c.expired, c.refreshed, c.bites)
		return c
	}
	unaccounted := func(c counts) float64 { return c.gained - c.refreshed - c.used - c.expired }

	weave := &proto.IndividualSimSettings{}
	if err := protojson.Unmarshal(raw, weave); err != nil {
		t.Fatalf("build: %v", err)
	}
	c := run(weave)
	if c.gained == 0 || c.bites == 0 || c.refreshed == 0 {
		t.Fatalf("the weave: the check needs procs, refreshes and Mongoose Bites")
	}
	if math.Abs(c.used-c.bites) > 1e-9 {
		t.Errorf("the weave: used %.2f a fight, Mongoose Bites %.2f; want equal", c.used, c.bites)
	}
	if left := unaccounted(c); left < -0.001 || left > 1.001 {
		t.Errorf("the weave: %.2f a fight unaccounted, want 0 to 1 (still up at the end)", left)
	}

	ranged := googleProto.Clone(weave).(*proto.IndividualSimSettings)
	mark := `{"spellId":` + strconv.Itoa(int(huntersMarkRank.ID)) + `}`
	ranged.Player.Rotation = resultsRotation(t, `{"type":"TypeAPL","priorityList":[`+
		`{"action":{"condition":{"not":{"val":{"auraIsActive":{"sourceUnit":{"type":"CurrentTarget"},"auraId":`+mark+`}}}},"castSpell":{"spellId":`+mark+`}}},`+
		`{"action":{"castSpell":{"spellId":{"spellId":`+strconv.Itoa(int(arcaneShotRank.ID))+`}}}}]}`)
	ranged.Player.DistanceFromTarget = 12
	c = run(ranged)
	if c.gained == 0 || c.expired == 0 || c.used != 0 || c.bites != 0 {
		t.Errorf("from range: %.2f procs, %.2f expired, %.2f used, %.2f bites; want procs that expire and none used", c.gained, c.expired, c.used, c.bites)
	}
	if left := unaccounted(c); left < 0.001 || left > 1.001 {
		t.Errorf("from range: %.2f a fight unaccounted, want above 0 (fights that end with it up) and at most 1", left)
	}
}

// Each mana per 5 source has its own mana row: a ring, Brilliant Mana Oil, Mageblood Elixir, Greater Blessing of
// Wisdom and Mana Spring Totem take the share of the mana per 5 regen that their mana per 5 is of the hunter's, and
// bonus stats, which no source names, fill the unnamed row. Without them the unnamed row stays empty.
func TestManaPerFiveRowsBySource(t *testing.T) {
	const ring, oil, mageblood = 19397, 20748, 20007
	type setup struct {
		without string
		bonus   float64
	}
	raid := func(s setup) *proto.Raid {
		gear := googleProto.Clone(core.GetGearSet("../../ui/specs/hunter/dps/gear_sets", "p1").GearSet).(*proto.EquipmentSpec)
		consumables := &proto.ConsumesSpec{}
		individual := &proto.IndividualBuffs{}
		party := &proto.PartyBuffs{}
		if s.without != "ring" {
			gear.Items[proto.ItemSlot_ItemSlotFinger1] = &proto.ItemSpec{Id: ring}
		}
		if s.without != "oil" {
			consumables.MhImbueId = oil
		}
		if s.without != "mageblood" {
			consumables.ElixirIds = []int32{mageblood}
		}
		if s.without != "wisdom" {
			individual.GreaterBlessingOfWisdom = true
		}
		if s.without != "spring" {
			party.ManaSpringTotem = proto.TristateEffect_TristateEffectRegular
		}
		bonus := make([]float64, stats.ProtoStatsLen)
		bonus[stats.MP5] = s.bonus
		player := &proto.Player{
			Name:               "Drinker",
			Race:               proto.Race_RaceOrc,
			Class:              proto.Class_ClassHunter,
			Equipment:          gear,
			Consumables:        consumables,
			Buffs:              individual,
			BonusStats:         &proto.UnitStats{Stats: bonus},
			TalentsString:      DefaultMMTalents,
			Spec:               DefaultOptions,
			Rotation:           resultsRotation(t, `{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":`+strconv.Itoa(int(arcaneShotRank.ID))+`}}}}]}`),
			DistanceFromTarget: 12,
		}
		return core.SinglePlayerRaidProto(player, party, &proto.RaidBuffs{}, &proto.Debuffs{})
	}
	// the mana per 5 the sim runs with, as the fight starts
	mp5 := func(s setup) float64 {
		sim := core.NewSim(&proto.RaidSimRequest{Raid: raid(s), Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{RandomSeed: 1}},
			simsignals.CreateSignals())
		sim.Reset()
		return sim.Raid.Parties[0].Players[0].GetCharacter().GetStat(stats.MP5)
	}
	rows := func(s setup) (map[string]float64, float64) {
		named := map[core.ActionID]string{
			{ItemID: ring}: "ring", {ItemID: oil}: "oil", {ItemID: mageblood}: "mageblood",
			core.ActionID{SpellID: 25918}.WithTag(-1): "wisdom", core.ActionID{SpellID: 10494}.WithTag(-1): "spring",
			{OtherID: proto.OtherAction_OtherActionManaRegen, Tag: 3}: "unnamed",
		}
		gains := map[string]float64{}
		total := 0.0
		for _, r := range runResultsSim(t, raid(s), core.MakeSingleTargetEncounter(0), 1).Resources {
			id := core.ProtoToActionID(r.Id)
			if r.Type != proto.ResourceType_ResourceTypeMana || r.Gain <= 0 || id.OtherID == proto.OtherAction_OtherActionManaRegen && id.Tag != 3 {
				continue
			}
			// every other mana gain here is mana per 5
			gains[named[id]] += r.Gain
			total += r.Gain
		}
		return gains, total
	}

	for _, bonus := range []float64{0, 10} {
		gains, total := rows(setup{bonus: bonus})
		if total <= 0 {
			t.Fatalf("bonus %.0f: no mana per 5 rows", bonus)
		}
		all := mp5(setup{bonus: bonus})
		t.Logf("bonus %.0f: %.1f mana per 5, rows %v", bonus, all, gains)
		// the rows hold the mana per 5 regen of the whole fight, no more and no less
		if want := all / 5 * core.LongDuration; math.Abs(total-want) > 1e-6*want {
			t.Errorf("bonus %.0f: rows add up to %.1f mana, want %.1f", bonus, total, want)
		}
		for _, source := range []string{"ring", "oil", "mageblood", "wisdom", "spring"} {
			want := (all - mp5(setup{without: source, bonus: bonus})) / all
			if want <= 0 {
				t.Fatalf("%s gives no mana per 5", source)
			}
			if got := gains[source] / total; math.Abs(got-want) > 1e-6 {
				t.Errorf("bonus %.0f: %s row %.4f of the mana per 5 regen, want %.4f", bonus, source, got, want)
			}
		}
		if got, want := gains["unnamed"]/total, bonus/all; math.Abs(got-want) > 1e-6 {
			t.Errorf("bonus %.0f: unnamed row %.4f of the mana per 5 regen, want %.4f", bonus, got, want)
		}
		if got := gains[""]; got != 0 {
			t.Errorf("bonus %.0f: %.1f mana per 5 in rows of other sources", bonus, got)
		}
	}
}
