// Adds the classic items (tools/database/arc_classic_items.json) to the committed database files
// (assets/database/db.bin, db.json) without regenerating from game data. ARC_ROUNDTRIP_ONLY=1 only reads and writes,
// to prove nothing else changes.
package main

import (
	"log"
	"os"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/tools/database"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	const dir = "assets/database"
	raw, err := os.ReadFile(dir + "/db.json")
	if err != nil {
		log.Fatal(err)
	}
	db := database.ReadDatabaseFromJson(string(raw))
	// the JSON reader leaves the preset encounters out
	full := &proto.UIDatabase{}
	if err := protojson.Unmarshal(raw, full); err != nil {
		log.Fatal(err)
	}
	db.Encounters = full.Encounters
	if os.Getenv("ARC_ROUNDTRIP_ONLY") == "" {
		before := len(db.Items)
		db.AddArcClassicData(true)
		// npc names an older importer left with AtlasLoot code in them
		for _, npc := range db.Npcs {
			npc.Name = database.AtlasLootNpcName(npc.Name)
		}
		log.Printf("items %d -> %d", before, len(db.Items))
	}
	db.WriteBinaryAndJson(dir+"/db.bin", dir+"/db.json")
}
