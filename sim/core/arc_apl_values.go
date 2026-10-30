package core

import (
	"fmt"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// APL values of this fork's own, for rotations to read the rules it plays by.

// How the swing timers are linked: 0 not linked, 1 melee swings restart Auto Shot, 2 both ways.
type APLValueSwingLink struct {
	DefaultAPLValueImpl
	unit *Unit
}

func (rot *APLRotation) newValueSwingLink(_ *proto.APLValueSwingLink, _ *proto.UUID) APLValue {
	return &APLValueSwingLink{unit: rot.unit}
}
func (value *APLValueSwingLink) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeInt
}
func (value *APLValueSwingLink) GetInt(_ *Simulation) int32 {
	aa := &value.unit.AutoAttacks
	switch {
	case !aa.MeleeResetsAutoShot:
		return 0
	case aa.AutoShotResetsMainHand:
		return 2
	}
	return 1
}
func (value *APLValueSwingLink) String() string {
	return "Swing Link"
}

// The melee attack is switched on.
type APLValueMeleeAttackOn struct {
	DefaultAPLValueImpl
	unit *Unit
}

func (rot *APLRotation) newValueMeleeAttackOn(_ *proto.APLValueMeleeAttackOn, _ *proto.UUID) APLValue {
	return &APLValueMeleeAttackOn{unit: rot.unit}
}
func (value *APLValueMeleeAttackOn) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeBool
}
func (value *APLValueMeleeAttackOn) GetBool(_ *Simulation) bool {
	return value.unit.AutoAttacks.meleeAttackOn
}
func (value *APLValueMeleeAttackOn) String() string {
	return "Melee Attack On"
}

// How long a weapon has been ready without swinging: its timer reached ready and no swing has gone since.
type APLValueSwingWaiting struct {
	DefaultAPLValueImpl
	weapon *WeaponAttack
}

func (rot *APLRotation) newValueSwingWaiting(config *proto.APLValueSwingWaiting, _ *proto.UUID) APLValue {
	aa := &rot.unit.AutoAttacks
	var weapon *WeaponAttack
	switch config.AutoType {
	case proto.APLValueAutoAttackType_MeleeAuto, proto.APLValueAutoAttackType_MainHandAuto:
		weapon = &aa.mh
	case proto.APLValueAutoAttackType_OffHandAuto:
		weapon = &aa.oh
	case proto.APLValueAutoAttackType_RangedAuto:
		weapon = &aa.ranged
	default:
		return nil
	}
	return &APLValueSwingWaiting{weapon: weapon}
}
func (value *APLValueSwingWaiting) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeDuration
}
func (value *APLValueSwingWaiting) GetDuration(sim *Simulation) time.Duration {
	if value.weapon.spell == nil {
		return 0
	}
	return max(0, sim.CurrentTime-value.weapon.readyAt)
}
func (value *APLValueSwingWaiting) String() string {
	return "Swing Waiting"
}

// Where the walk under way or decided is headed; with no walk, where the unit stands.
type APLValueWalkTarget struct {
	DefaultAPLValueImpl
	unit *Unit
}

func (rot *APLRotation) newValueWalkTarget(_ *proto.APLValueWalkTarget, _ *proto.UUID) APLValue {
	return &APLValueWalkTarget{unit: rot.unit}
}
func (value *APLValueWalkTarget) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeFloat
}
func (value *APLValueWalkTarget) GetFloat(sim *Simulation) float64 {
	unit := value.unit
	walk := &unit.arcWalk
	switch {
	case walk.pending:
		return walk.pendingRange
	case walk.queued:
		return walk.queuedRange
	case unit.movementAction != nil:
		return unit.arcWalkTarget()
	}
	unit.UpdatePosition(sim, false)
	return unit.DistanceFromTarget
}
func (value *APLValueWalkTarget) String() string {
	return "Walk Target"
}

// A walk is decided but not under way yet.
type APLValueWalkPending struct {
	DefaultAPLValueImpl
	unit *Unit
}

func (rot *APLRotation) newValueWalkPending(_ *proto.APLValueWalkPending, _ *proto.UUID) APLValue {
	return &APLValueWalkPending{unit: rot.unit}
}
func (value *APLValueWalkPending) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeBool
}
func (value *APLValueWalkPending) GetBool(_ *Simulation) bool {
	return value.unit.arcWalk.pending || value.unit.arcWalk.queued
}
func (value *APLValueWalkPending) String() string {
	return "Walk Pending"
}

// How long the regen takes to reach the given mana, spending none meanwhile.
type APLValueManaTimeToTarget struct {
	DefaultAPLValueImpl
	unit       *Unit
	targetMana APLValue
}

func (rot *APLRotation) newValueManaTimeToTarget(config *proto.APLValueManaTimeToTarget, uuid *proto.UUID) APLValue {
	if !rot.unit.HasManaBar() {
		rot.ValidationMessageByUUID(uuid, proto.LogLevel_Warning, "%s does not use Mana", rot.unit.Label)
		return nil
	}
	targetMana := rot.coerceTo(rot.newAPLValue(config.TargetMana), proto.APLValueType_ValueTypeFloat)
	if targetMana == nil {
		return nil
	}
	return &APLValueManaTimeToTarget{unit: rot.unit, targetMana: targetMana}
}
func (value *APLValueManaTimeToTarget) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeDuration
}
func (value *APLValueManaTimeToTarget) GetDuration(sim *Simulation) time.Duration {
	return value.unit.TimeUntilManaRegen(sim, value.targetMana.GetFloat(sim))
}
func (value *APLValueManaTimeToTarget) String() string {
	return fmt.Sprintf("Mana Time To(%s)", value.targetMana)
}

// How long ago the spell last went out; never cast this fight, longer than any fight.
type APLValueSpellTimeSinceLastCast struct {
	DefaultAPLValueImpl
	spell *Spell
}

func (rot *APLRotation) newValueSpellTimeSinceLastCast(config *proto.APLValueSpellTimeSinceLastCast, _ *proto.UUID) APLValue {
	spell := rot.GetAPLSpell(config.SpellId)
	if spell == nil {
		return nil
	}
	return &APLValueSpellTimeSinceLastCast{spell: spell}
}
func (value *APLValueSpellTimeSinceLastCast) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeDuration
}
func (value *APLValueSpellTimeSinceLastCast) GetDuration(sim *Simulation) time.Duration {
	if value.spell.casts == 0 {
		return NeverExpires
	}
	return sim.CurrentTime - value.spell.lastCastAt
}
func (value *APLValueSpellTimeSinceLastCast) String() string {
	return fmt.Sprintf("Time Since Last Cast(%s)", value.spell.ActionID)
}
