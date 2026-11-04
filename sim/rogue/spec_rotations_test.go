package rogue

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/wowsims/forever/sim/core"
)

var castSpellID = regexp.MustCompile(`"castSpell":\{"spellId":\{(?:"rank":\d+,)?"spellId":(\d+)`)

// Every spell a spec rotation casts is one the rogue registers with that build's talents: an id the engine does
// not know (a book rank, another rank of the spell) drops its line without a word.
func TestSpecRotationsCastOnlyRegisteredSpells(t *testing.T) {
	for _, c := range []struct{ apl, talents string }{
		{"mutilate", "00530320241151031-30230302020113"},
		{"combat_backstab", "005303202-3023030202012221251-005"},
		{"combat_sinister", "005303202001-3203030202012321551"},
		{"hemorrhage", "00530321-320001-015203031321021151"},
	} {
		text, err := os.ReadFile("../../ui/specs/rogue/dps/apls/" + c.apl + ".apl.json")
		if err != nil {
			t.Fatal(err)
		}
		_, rogue, _ := newRogueSim(t, c.talents)
		matches := castSpellID.FindAllSubmatch(text, -1)
		if len(matches) == 0 {
			t.Fatalf("%s casts nothing", c.apl)
		}
		for _, m := range matches {
			id, _ := strconv.Atoi(string(m[1]))
			if rogue.GetSpell(core.ActionID{SpellID: int32(id)}) == nil {
				t.Errorf("%s casts spell %d, which the rogue does not register", c.apl, id)
			}
		}
	}
}

// The Mutilate rotation as the runner measured it best at 20k on the 2 min reference fight: no Ambush from Stealth
// (opening with Mutilate is +4.2), Slice and Dice goes up at two points, and Shank comes ahead of Eviscerate.
func TestMutilateRotationKeepsItsMeasuredOrder(t *testing.T) {
	text, err := os.ReadFile("../../ui/specs/rogue/dps/apls/mutilate.apl.json")
	if err != nil {
		t.Fatal(err)
	}
	var order []int
	list := text[bytes.Index(text, []byte(`"priorityList"`)):]
	for _, m := range castSpellID.FindAllSubmatch(list, -1) {
		id, _ := strconv.Atoi(string(m[1]))
		order = append(order, id)
	}
	at := func(id int) int {
		for i, v := range order {
			if v == id {
				return i
			}
		}
		t.Fatalf("the rotation never casts %d", id)
		return -1
	}
	if order[0] != 6774 || bytes.Contains(text, []byte(`"spellId":11269`)) {
		t.Errorf("the rotation opens with %d or casts Ambush, want Slice and Dice 6774 first and no Ambush", order[0])
	}
	if at(1329549) > at(11300) {
		t.Error("Shank comes after Eviscerate")
	}
	snd := regexp.MustCompile(`"castSpell":\{"spellId":\{"rank":2,"spellId":6774\}\},"condition":\{"and":\{"vals":\[\{"cmp":\{"op":"OpGe","lhs":\{"currentComboPoints":\{\}\},"rhs":\{"const":\{"val":"2"\}\}`)
	if !snd.Match(text) {
		t.Error("Slice and Dice no longer waits for two combo points")
	}
}

// Both Combat rotations as the runner measured them best on the 2 min reference fight: Blade Flurry on cooldown right
// after the opener, ahead of Slice and Dice; Backstab opens with Ambush, Sinister Strike (a sword in the main hand) with
// no opener (Garrote from Stealth is -5.8).
func TestCombatRotationsKeepTheirMeasuredOrder(t *testing.T) {
	for _, c := range []struct {
		apl             string
		opener, builder int
	}{
		{"combat_backstab", 11269, 11281},
		{"combat_sinister", 0, 11294},
	} {
		text, err := os.ReadFile("../../ui/specs/rogue/dps/apls/" + c.apl + ".apl.json")
		if err != nil {
			t.Fatal(err)
		}
		var order []int
		for _, m := range castSpellID.FindAllSubmatch(text[bytes.Index(text, []byte(`"priorityList"`)):], -1) {
			id, _ := strconv.Atoi(string(m[1]))
			order = append(order, id)
		}
		if c.opener == 0 {
			order = append([]int{0}, order...)
		}
		if len(order) < 3 || order[0] != c.opener || order[1] != 13877 || order[2] != 6774 {
			t.Errorf("%s starts %v, want opener %d, Blade Flurry, Slice and Dice", c.apl, order, c.opener)
		}
		if !bytes.Contains(text, []byte(`{"action":{"castSpell":{"spellId":{"spellId":13877}}}}`)) {
			t.Errorf("%s waits to cast Blade Flurry", c.apl)
		}
		if order[len(order)-1] != c.builder {
			t.Errorf("%s ends with %d, want the builder %d", c.apl, order[len(order)-1], c.builder)
		}
	}
}

// The Hemorrhage rotation as the runner measured it best at 20k: Garrote and Ambush whenever they can
// be used, Rupture ahead of the finishers, Vanish and Preparation for more openers, with Ghostly Strike pressed when
// Garrote has under 3 s left or was just put up, so its buff renews Garrote: Garrote is up 91% of the fight, not 70%.
// Rupture waits for three points, Hemorrhage for 45 energy.
func TestHemorrhageRotationKeepsItsMeasuredOrder(t *testing.T) {
	text, err := os.ReadFile("../../ui/specs/rogue/dps/apls/hemorrhage.apl.json")
	if err != nil {
		t.Fatal(err)
	}
	list := text[bytes.Index(text, []byte(`"priorityList"`)):]
	var order []int
	for _, m := range castSpellID.FindAllSubmatch(list, -1) {
		id, _ := strconv.Atoi(string(m[1]))
		order = append(order, id)
	}
	at := func(id int) int {
		for i, v := range order {
			if v == id {
				return i
			}
		}
		return -1
	}
	for _, id := range []int{11290, 11269, 11275, 11300, 1857, 14278, 16511, 14185} {
		if at(id) < 0 {
			t.Errorf("the rotation never casts %d", id)
		}
	}
	if !(at(11290) < at(11269) && at(11269) < at(11275) && at(11275) < at(11300) && at(11300) < at(1857)) {
		t.Errorf("order %v: want Garrote, Ambush, Rupture, Eviscerate, then Vanish", order)
	}
	ghostly := regexp.MustCompile(`"castSpell":\{"spellId":\{"spellId":14278\}\},"condition":\{"or":\{"vals":\[\{"cmp":\{"op":"OpLt","lhs":\{"dotRemainingTime":\{"spellId":\{"spellId":11290\}\}\},"rhs":\{"const":\{"val":"3s"\}\}`)
	if !ghostly.Match(list) {
		t.Error("Ghostly Strike no longer waits for Garrote to have under 3 s left")
	}
	if !bytes.Contains(list, []byte(`{"cmp":{"op":"OpGe","lhs":{"currentComboPoints":{}},"rhs":{"const":{"val":"3"}}}}`)) {
		t.Error("Rupture no longer waits for three combo points")
	}
	if !bytes.Contains(list, []byte(`"castSpell":{"spellId":{"spellId":16511}},"condition":{"and":{"vals":[{"cmp":{"op":"OpGe","lhs":{"currentEnergy":{}},"rhs":{"const":{"val":"45"}}}}`)) {
		t.Error("Hemorrhage no longer pools to 45 energy")
	}
}
