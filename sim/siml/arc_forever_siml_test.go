package siml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	googleProto "google.golang.org/protobuf/proto"
)

// A card with every field, and text no ident or note could hold, reads back from its //@ lines, in a group and
// in prepull too.
func TestEveryCardFieldRoundTrips(t *testing.T) {
	names, err := DefaultNames()
	if err != nil {
		t.Fatal(err)
	}
	card := &proto.APLSimpleView{
		Shown: true,
		Text:  "**Move in** to {Spot} yd \"now\"\nthen {Gap} s\tlater ",
		Title: "Step in",
		Boxes: []*proto.APLSimpleBox{
			{Label: "Spot", Numbers: []string{"move/rangeFromTarget", "condition/cmp/rhs"}, Unit: "yd"},
			{Label: "Bare"},
			{Label: "Unit only", Unit: "s"},
		},
		Values: []*proto.APLSimpleValue{{Name: "Gap", Expr: "Main hand swing - Auto Shot swing"}},
		Texts:  []*proto.APLSimpleText{{When: "Gap >= 0.5", Text: "**Wait**"}, {When: "", Text: ""}},
	}
	row := func(simple *proto.APLSimpleView) *proto.APLListItem {
		return &proto.APLListItem{Notes: "a note", Simple: simple, Action: &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{}}}}
	}
	rotation := &proto.APLRotation{
		Type:         proto.APLRotation_TypeAPL,
		PriorityList: []*proto.APLListItem{row(card), row(&proto.APLSimpleView{}), row(nil)},
		Groups:       []*proto.APLGroup{{Name: "g", Actions: []*proto.APLListItem{row(card)}}},
		PrepullActions: []*proto.APLPrepullAction{{
			DoAtValue: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "-2s"}}},
			Action:    &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{}}},
			Simple:    card,
		}},
	}
	settings := &proto.IndividualSimSettings{Player: &proto.Player{Rotation: rotation}}
	printed := PrintRequest(&proto.SimlPrintRequest{Section: "apl", Spec: proto.Spec_SpecHunter, Settings: settings}, names)
	if printed.Text == "" || strings.Contains(printed.Text, `"simple"`) {
		t.Fatalf("the card does not print as //@ lines: %s %v\n%s", printed.Error, printed.Diagnostics, printed.Text)
	}
	read := ParseRequest(&proto.SimlParseRequest{Path: "x.apl.siml", Text: printed.Text}, names)
	if read.Settings == nil {
		t.Fatalf("the text does not read: %s %v\n%s", read.Error, read.Diagnostics, printed.Text)
	}
	if !googleProto.Equal(read.Settings.Player.Rotation, rotation) {
		t.Errorf("the rotation reads back changed from:\n%s", printed.Text)
	}

	// A card above a raw JSON action is its row's card, as notes are (SPEC.md §6.3).
	text := "siml 1\nspec hunter\n\npriority:\n  //@card \"Raw\"\n  `{\"autocastOtherCooldowns\":{}}`\n"
	raw := ParseRequest(&proto.SimlParseRequest{Path: "x.apl.siml", Text: text}, names)
	if raw.Settings == nil || raw.Settings.Player.Rotation.PriorityList[0].Simple.GetText() != "Raw" {
		t.Errorf("a raw row drops its card: %s %v", raw.Error, raw.Diagnostics)
	}
}

// A //@ line that is not above a rotation row, or that no card form reads, is an error.
func TestBadCardLines(t *testing.T) {
	names, err := DefaultNames()
	if err != nil {
		t.Fatal(err)
	}
	for text, code := range map[string]string{
		"siml 1\nspec hunter\n\npriority:\n  //@card \"a\"\n\n  autocast_other_cooldowns\n":                    "misplaced-card",
		"siml 1\nspec hunter\n\npriority:\n  autocast_other_cooldowns\n  //@card \"a\"\n":                        "misplaced-card",
		"siml 1\nspec hunter\n\npriority:\n  //@card shown\n  autocast_other_cooldowns\n":                         "bad-card",
		"siml 1\nspec hunter\n\npriority:\n  //@card \"a\"\n  //@card \"b\"\n  autocast_other_cooldowns\n":        "bad-card",
		"siml 1\nspec hunter\n\npriority:\n  //@box \"a\" at \"x\", unit \"yd\"\n  autocast_other_cooldowns\n":   "bad-card",
		"siml 1\nspec hunter\n\npriority:\n  //@when \"a\"\n  autocast_other_cooldowns\n":                         "bad-card",
		"siml 1\nspec hunter\n\npriority:\n  //@card \"open\n  autocast_other_cooldowns\n":                         "bad-card",
	} {
		read := ParseRequest(&proto.SimlParseRequest{Path: "x.apl.siml", Text: text}, names)
		found := false
		for _, d := range read.Diagnostics {
			found = found || d.Code == code
		}
		if !found {
			t.Errorf("%q gives no %s: %v", text, code, read.Diagnostics)
		}
	}
}

// Every hunter build's rotation exports with its cards as //@ lines, no row as raw JSON, and reads back the same.
func TestBuildRotationsExportWithTheirCards(t *testing.T) {
	names, err := DefaultNames()
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("../../ui/specs/hunter/dps/*builds/*.build.json")
	if len(files) < 20 {
		t.Fatalf("found %d build files", len(files))
	}
	total := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		settings := &proto.IndividualSimSettings{}
		if err := protojson.Unmarshal(data, settings); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		name := filepath.Base(file)
		rotation := &proto.IndividualSimSettings{Player: &proto.Player{Rotation: settings.Player.Rotation}}
		printed := PrintRequest(&proto.SimlPrintRequest{Section: "apl", Spec: proto.Spec_SpecHunter, Settings: rotation}, names)
		if printed.Text == "" {
			t.Fatalf("%s: the print fails: %s %v", name, printed.Error, printed.Diagnostics)
		}
		cards := 0
		for _, line := range strings.Split(printed.Text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "{") {
				t.Errorf("%s: a row prints as raw JSON: %.120s", name, line)
			}
			if strings.HasPrefix(line, "//@card ") {
				cards++
			}
		}
		want := 0
		for _, item := range settings.Player.Rotation.PriorityList {
			if item.Simple != nil {
				want++
			}
		}
		for _, item := range settings.Player.Rotation.PrepullActions {
			if item.Simple != nil {
				want++
			}
		}
		for _, group := range settings.Player.Rotation.Groups {
			for _, item := range group.Actions {
				if item.Simple != nil {
					want++
				}
			}
		}
		if cards != want {
			t.Errorf("%s: %d //@card lines for %d cards", name, cards, want)
		}
		total += want
		read := ParseRequest(&proto.SimlParseRequest{Path: "x.apl.siml", Text: printed.Text}, names)
		if read.Settings == nil {
			t.Fatalf("%s: the text does not read: %s %v", name, read.Error, read.Diagnostics)
		}
		// A spell name reads back with its entry's rank (SPEC.md §12.4), which Normalize clears on both sides.
		got := Normalize(&proto.IndividualSimSettings{Player: &proto.Player{Rotation: read.Settings.Player.Rotation}}, names, proto.Spec_SpecHunter)
		if !googleProto.Equal(got, Normalize(rotation, names, proto.Spec_SpecHunter)) {
			t.Errorf("%s: the rotation reads back changed", name)
		}
	}
	if total < 300 {
		t.Errorf("the builds hold %d cards", total)
	}
}

// The rotation's walk-in-starts-the-attack switch is ours; a SIML export that dropped it would change the rotation.
func TestManualStartAttackRoundTrips(t *testing.T) {
	names, err := DefaultNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, on := range []bool{true, false} {
		settings := &proto.IndividualSimSettings{Player: &proto.Player{Rotation: &proto.APLRotation{
			Type:              proto.APLRotation_TypeAPL,
			ManualStartAttack: on,
			PriorityList:      []*proto.APLListItem{{Action: &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{}}}}},
		}}}
		printed := PrintRequest(&proto.SimlPrintRequest{Section: "apl", Spec: proto.Spec_SpecHunter, Settings: settings}, names)
		if printed.Text == "" {
			t.Fatalf("manual start attack %v: the print fails: %s %v", on, printed.Error, printed.Diagnostics)
		}
		read := ParseRequest(&proto.SimlParseRequest{Path: "x.apl.siml", Text: printed.Text}, names)
		if read.Settings == nil {
			t.Fatalf("manual start attack %v: the text does not read: %s %v", on, read.Error, read.Diagnostics)
		}
		if got := read.Settings.Player.Rotation.ManualStartAttack; got != on {
			t.Errorf("manual start attack %v reads back as %v from:\n%s", on, got, printed.Text)
		}
	}
}
