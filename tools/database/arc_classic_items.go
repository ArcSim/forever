package database

import (
	_ "embed"

	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// Items the Forever client data does not carry, copied from the classic item data by refork/gen_classic_items.js,
// with the random suffixes, zones, npcs and factions they name. Stats only: procs and use effects come from the sim's
// item effect registrations (sim/common/forever/arc_classic_item_effects.go).
//
//go:embed arc_classic_items.json
var arcClassicJSON []byte

var ArcClassicData = func() *proto.UIDatabase {
	db := &proto.UIDatabase{}
	if err := protojson.Unmarshal(arcClassicJSON, db); err != nil {
		panic("arc_classic_items.json: " + err.Error())
	}
	return db
}()

var ArcClassicItems = ArcClassicData.Items

// Adds the classic items and what they name; the beta's own suffixes, zones, npcs and factions are kept. With replace,
// an item overwrites what the database holds under its id (an earlier copy of itself, so applying twice changes
// nothing); without it, game data that carries the item wins.
func (db *WowDatabase) AddArcClassicData(replace bool) {
	for _, item := range ArcClassicData.Items {
		if _, ok := db.Items[item.Id]; replace || !ok {
			db.Items[item.Id] = item
		}
	}
	for _, suffix := range ArcClassicData.RandomSuffixes {
		db.RandomSuffixes[suffix.Id] = suffix
	}
	for _, zone := range ArcClassicData.Zones {
		if _, ok := db.Zones[zone.Id]; !ok {
			db.Zones[zone.Id] = zone
		}
	}
	for _, npc := range ArcClassicData.Npcs {
		if _, ok := db.Npcs[npc.Id]; !ok {
			db.Npcs[npc.Id] = npc
		}
	}
	for _, faction := range ArcClassicData.Factions {
		if _, ok := db.Factions[faction.Id]; !ok {
			db.Factions[faction.Id] = faction
		}
	}
}
