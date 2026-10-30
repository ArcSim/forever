package database

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

func arcClassicItem(t *testing.T, id int32) *proto.UIItem {
	t.Helper()
	for _, item := range ArcClassicItems {
		if item.Id == id {
			return item
		}
	}
	t.Fatalf("item %d is not in arc_classic_items.json", id)
	return nil
}

func TestArcClassicDataCarriesTheClassicItems(t *testing.T) {
	if n := len(ArcClassicItems); n < 1600 {
		t.Fatalf("%d classic items, want the alpha's 1,600+", n)
	}

	// Barbarous Blade: 60 attack power once (not again as ranged), +1% crit as 14 rating, the Glaive's damage
	blade := arcClassicItem(t, 18520)
	scaling := blade.ScalingOptions[0]
	if scaling.Stats[14] != 60 || scaling.Stats[15] != 0 || scaling.Stats[18] != 14 || len(scaling.Stats) != 2 {
		t.Errorf("Barbarous Blade stats %v, want attack power 60 and crit rating 14 only", scaling.Stats)
	}
	if scaling.WeaponDamageMin != 138 || scaling.WeaponDamageMax != 207 || blade.WeaponSpeed != 3.2 || blade.HandType != proto.HandType_HandTypeTwoHand {
		t.Errorf("Barbarous Blade weapon %v-%v at %v, hand %v", scaling.WeaponDamageMin, scaling.WeaponDamageMax, blade.WeaponSpeed, blade.HandType)
	}

	// a class limit and a ranged weapon type in the beta's numbering
	if chest := arcClassicItem(t, 16845); len(chest.ClassAllowlist) != 1 || chest.ClassAllowlist[0] != proto.Class_ClassHunter {
		t.Errorf("Giantstalker's Breastplate classes %v, want hunter", chest.ClassAllowlist)
	}
	if crossbow := arcClassicItem(t, 20599); crossbow.RangedWeaponType != proto.RangedWeaponType_RangedWeaponTypeCrossbow {
		t.Errorf("Polished Ironwood Crossbow ranged type %v", crossbow.RangedWeaponType)
	}

	// a random suffix keeps its flat stats: the item's points are 10000, so allocation x 10000 / 10000
	ring := arcClassicItem(t, 862)
	if ring.ScalingOptions[0].RandPropPoints != 10000 || len(ring.RandomSuffixOptions) == 0 {
		t.Fatalf("Runed Ring points %d, %d suffixes", ring.ScalingOptions[0].RandPropPoints, len(ring.RandomSuffixOptions))
	}
	suffixes := map[int32]*proto.ItemRandomSuffix{}
	for _, suffix := range ArcClassicData.RandomSuffixes {
		suffixes[suffix.Id] = suffix
	}
	for _, id := range ring.RandomSuffixOptions {
		if suffixes[id] == nil {
			t.Errorf("Runed Ring names suffix %d, which the data does not carry", id)
		}
	}
	if intellect := suffixes[9000213]; intellect == nil || intellect.Stats[proto.Stat_StatIntellect] != 13 {
		t.Errorf("suffix 9000213 (of Intellect) is %v, want +13 Intellect", intellect)
	}

	// a drop from a world boss that roams has no zone, but names its npc
	if footguards := arcClassicItem(t, 20629); footguards.Sources[0].GetDrop().GetNpcId() != 14888 {
		t.Errorf("Malignant Footguards source %v, want Lethon", footguards.Sources)
	}
	named := false
	for _, npc := range ArcClassicData.Npcs {
		named = named || (npc.Id == 14888 && npc.Name == "Lethon")
	}
	if !named {
		t.Error("Lethon is not among the npcs the data names")
	}
}

func TestAtlasLootNpcNameEndsAtItsOwnBracket(t *testing.T) {
	for matched, want := range map[string]string{
		`Kormok"].." - "..format(AL["Tier %s Sets`: "Kormok",
		"Warchief Rend Blackhand":                   "Warchief Rend Blackhand",
		`Eric "The Swift"`:                          `Eric "The Swift"`,
	} {
		if got := AtlasLootNpcName(matched); got != want {
			t.Errorf("AtlasLootNpcName(%q) = %q, want %q", matched, got, want)
		}
	}
}

func TestAddArcClassicDataLetsGameDataWinOnlyWithoutReplace(t *testing.T) {
	newDB := func() *WowDatabase {
		db := NewWowDatabase()
		db.Items[18520] = &proto.UIItem{Id: 18520, Name: "Barbarous Blade from game data"}
		return db
	}

	kept := newDB()
	kept.AddArcClassicData(false)
	if kept.Items[18520].Name != "Barbarous Blade from game data" {
		t.Errorf("without replace the game data's copy should stay, got %q", kept.Items[18520].Name)
	}

	replaced := newDB()
	replaced.AddArcClassicData(true)
	count := len(replaced.Items)
	replaced.AddArcClassicData(true)
	if replaced.Items[18520].Name != "Barbarous Blade" || len(replaced.Items) != count {
		t.Errorf("with replace the classic copy should win and a second pass add nothing: %q, %d -> %d items", replaced.Items[18520].Name, count, len(replaced.Items))
	}
}
