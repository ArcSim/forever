package hunter

// A bullet and the damage per second it adds to the ranged weapon, as its tooltip reads.
type bullet struct {
	Name   string
	ItemID int32
	DPS    float64
}

// The bullets the Forever client's items list, keyed by the Go identifier proto.HunterOptions_Ammo
// names them by. The generated arrows come from the damage table by item level and quality, which
// the bullets share with the arrow of the same level and quality; Mithril Gyro-Shot has no such arrow,
// so its 15 is the tooltip's.
var bullets = map[string]bullet{
	"SolidShot":         {Name: "Solid Shot", ItemID: 3033, DPS: 7.5},
	"AccurateSlugs":     {Name: "Accurate Slugs", ItemID: 11284, DPS: 13},
	"MithrilGyroShot":   {Name: "Mithril Gyro-Shot", ItemID: 10513, DPS: 15},
	"IceThreadedBullet": {Name: "Ice Threaded Bullet", ItemID: 19317, DPS: 16.5},
	"ThoriumShells":     {Name: "Thorium Shells", ItemID: 15997, DPS: 17.5},
	"SwiftstrikeShot":   {Name: "Swiftstrike Shot", ItemID: 274388, DPS: 24.5},
}

// The damage per second the chosen arrow or bullet adds.
func ammoDPS(name string) float64 {
	if a, ok := arrows[name]; ok {
		return a.DPS
	}
	return bullets[name].DPS
}
