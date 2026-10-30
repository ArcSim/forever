package hunter

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/spelldata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Aspect of the Hawk rank 7 and Serpent Sting rank 9 are taught only by Ruins of Ahn'Qiraj books (Guide: Aspect of the
// Hawk VII, Guide: Serpent Sting IX); a pre-raid hunter has the trainer's top ranks (core.ForeverTrainerRanks).
const (
	topTrainerRankHawk         = 6
	topTrainerRankSerpentSting = 8
)

func trainerRank(l spelldata.Ladder, top int32) *spelldata.Spell {
	if core.ForeverTrainerRanks && l.Len() > top {
		return l.Rank(top)
	}
	return l.Highest()
}

// rotation spell -> the spell the hunter casts instead, so rotations saved with the book ranks still cast
var rotationSpellMigrations = map[int32]struct{ spellID, rank int32 }{
	25296: {14322, topTrainerRankHawk},
	25295: {13555, topTrainerRankSerpentSting},
}

func migrateBookRanks(rotation *proto.APLRotation) {
	if rotation != nil && core.ForeverTrainerRanks {
		migrateActionIDs(rotation.ProtoReflect())
	}
}

func migrateActionIDs(m protoreflect.Message) {
	if id, ok := m.Interface().(*proto.ActionID); ok {
		if to, found := rotationSpellMigrations[id.GetSpellId()]; found {
			id.RawId = &proto.ActionID_SpellId{SpellId: to.spellID}
			id.Rank = to.rank
		}
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
			return true
		}
		switch {
		case fd.IsList():
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				migrateActionIDs(list.Get(i).Message())
			}
		case fd.IsMap():
			if fd.MapValue().Kind() == protoreflect.MessageKind {
				v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
					migrateActionIDs(mv.Message())
					return true
				})
			}
		default:
			migrateActionIDs(v.Message())
		}
		return true
	})
}
