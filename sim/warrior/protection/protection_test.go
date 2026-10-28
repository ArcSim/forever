package protection

import (
	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common" // imported to get item effects included.
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"

	"fmt"
	"testing"
)

func init() {
	RegisterProtectionWarrior()
	common.RegisterAllEffects()
}

func TestProtectionWarrior(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{protectionWarriorSuite()}))
}

func BenchmarkProtectionWarrior(b *testing.B) {
	core.CharacterBenchmark(b, protectionWarriorSuite())
}

func protectionWarriorSuite() core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:      proto.Class_ClassWarrior,
		Race:       proto.Race_RaceOrc,
		OtherRaces: []proto.Race{proto.Race_RaceHuman},
		GearSet:    core.GetGearSet("../../../ui/specs/warrior/protection/gear_sets", "prebis"),
		OtherGearSets: []core.GearSetCombo{
			core.GetGearSet("../../../ui/specs/warrior/protection/gear_sets", "prebis_threat"),
			core.GetGearSet("../../../ui/specs/warrior/protection/gear_sets", "p1_bis"),
		},
		Talents:          DefaultProtectionTalents,
		Consumables:      DefaultConsumables,
		SpecOptions:      core.SpecOptionsCombo{Label: "Protection", SpecOptions: DefaultOptions},
		StartingDistance: 0,
		Profession1:      proto.Profession_Engineering,
		Profession2:      proto.Profession_Blacksmithing,

		Rotation: core.GetAplRotation("../../../ui/specs/warrior/protection/apls", "default"),
		OtherRotations: []core.RotationCombo{
			bookRanksRotation("default", "Manual: Heroic Strike IX", "Manual: Battle Shout VII", "Manual: Revenge VI"),
		},

		IndividualBuffs: core.FullTankIndividualBuffs,

		IsTank:          true,
		InFrontOfTarget: true,

		ItemFilter: core.ItemFilter{
			ArmorType: proto.ArmorType_ArmorTypePlate,
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeFist,
				proto.WeaponType_WeaponTypeMace,
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypeAxe,
				proto.WeaponType_WeaponTypeShield,
			},
			HandTypes: []proto.HandType{
				proto.HandType_HandTypeMainHand,
				proto.HandType_HandTypeOffHand,
				proto.HandType_HandTypeOneHand,
			},
		},
	}
}

// The preset with its book rank variables on, so the ranks from the class books reach the results.
func bookRanksRotation(preset string, variables ...string) core.RotationCombo {
	rotation := core.GetAplRotation("../../../ui/specs/warrior/protection/apls", preset).Rotation
	for _, name := range variables {
		found := false
		for _, variable := range rotation.ValueVariables {
			if variable.Name == name && variable.Value.GetConst() != nil {
				variable.Value.GetConst().Val = "true"
				found = true
			}
		}
		if !found {
			panic(fmt.Sprintf("the %s preset has no const variable %q", preset, name))
		}
	}
	return core.RotationCombo{Label: preset + "_book_ranks", Rotation: rotation}
}

var DefaultOptions = &proto.Player_ProtectionWarrior{
	ProtectionWarrior: &proto.ProtectionWarrior{
		Options: &proto.ProtectionWarrior_Options{
			ClassOptions: &proto.WarriorOptions{
				StartingRage:   0,
				UseBattleShout: true,
				DefaultStance:  proto.WarriorStance_WarriorStanceDefensive,
				QueueDelay:     250,
				StanceSnapshot: true,
				HasBsT2:        true,
			},
		},
	},
}

var DefaultProtectionTalents = "35301003002-03-050530120301011351"

var DefaultConsumables = &proto.ConsumesSpec{
	OffensivePotId: 250943, // Major Frenzy Potion
	FoodId:         20452,  // Smoked Desert Dumplings
	ExplosiveId:    260817, // EZ-Thro Dark Bomb
	MhImbueId:      12404,  // Dense Sharpening Stone
	ConjuredId:     9421,   // Major Healthstone
	// Elixir of Ferocity, Elixir of the Mongoose, Winterfall Firewater, Elixir of Greater Defense, Elixir of the Phalanx
	ElixirIds: []int32{250350, 13452, 12820, 13445, 250329},
	ExtraIds:  []int32{10646, 17048}, // Goblin Sapper Charge, Rumsey Rum
}
