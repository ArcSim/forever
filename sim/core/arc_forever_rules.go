package core

import (
	"os"
	"strings"
	"time"
)

// Rules from Arc's Forever engine, each a switch so a comparison can turn one back to this engine's rule.
var (
	// No 1.8% aura crit suppression against targets three or more levels above.
	ForeverNoAuraCritSuppression = false
	// A walk into melee turns the melee attack on as it starts, not on arrival.
	ForeverWalkInStartsAttack = true
	// Traps can only be dropped within 6 yd of the target, and catch it from there.
	ForeverTrapReach = true
	// Attack power on gear raises ranged attack power too.
	ForeverGearAPIsRanged = true
	// Hunter's Mark gives 143 ranged attack power, not its rank's 71. Trueshot Aura takes its top rank's 50.
	ForeverMarkAndTrueshot = true
	// Pets regenerate focus twice as fast.
	ForeverPetFocusDoubled = true
	// Traps take 0.132 of the higher attack power on Immolation Trap's burn and Explosive Trap's first hit.
	ForeverTrapAP = true
	// Serpent Sting ticks for its rank's base plus 3.0% of ranged attack power, read when it lands.
	ForeverSerpentSting = true
	// Arc's walk rules (arc_forever_movement.go): a move reaction before each walk, a walk out decided on the way in,
	// Auto Shot 0.6 s after a walk ends.
	ForeverWalkRules = true
	// The rotation picks the next spell when it can go, at the GCD's end, not up to 0.4 s before it.
	ForeverNoSpellQueue = true
	// A Strict Sequence starts only with the GCD free and its spells ready now.
	ForeverStrictSequenceNow = true
	// The off hand takes the swing retry too, like the main hand.
	ForeverOffHandRetry = true
	// Windfury Totem: 100 ms between procs.
	ForeverWindfuryICD = true
	// A trap's burn lands 1.2 s after the cast.
	ForeverTrapArm = true
	// A trap's hit roll adds Survival Tactics (5% a point) to the hunter's spell hit.
	ForeverTrapHit = true
	// An area around the caster reaches a target whose hitbox edge is inside its radius.
	ForeverAreaReach = true
	// A rotation that moves is looked at every 50 ms through the GCD, from the spell that starts the GCD on.
	ForeverGCDPoll = true
	// Only the consumables the build names are carried, not every potion and conjured item the page lists as usable.
	ForeverChosenConsumablesOnly = true
	// A hunter knows the trainer's top ranks: Aspect of the Hawk 7 and Serpent Sting 9 are Ruins of Ahn'Qiraj books.
	ForeverTrainerRanks = true
)

// Distances run centre to centre, and the 11 yd ranged spot is 8 yd between hitbox edges, so an area around the
// caster reaches 3 yd past its radius.
func ForeverAreaReachBonus() float64 {
	return TernaryFloat64(ForeverAreaReach, 3, 0)
}

// How often a main hand that came ready out of melee range tries the swing again, counted from the later of
// its timer running out and the melee attack turning on.
const MeleeAutoRetryInterval = time.Millisecond * 500

// How long after a unit gets in melee range the server sees it there.
const MeleeArrivalDelay = time.Millisecond * 50

// ARC_OFF lists rules to turn back to this engine's behaviour for a comparison: aura, walkin, trap, gearap, marktsa, petfocus, trapap, sting, walk, seq, ohretry, wficd, traparm, traphit, area, gcdpoll, consumes, queue, ranks.
func init() {
	// ARC_ON=steps turns whole-yard walks back on for a comparison
	if strings.Contains(os.Getenv("ARC_ON"), "steps") {
		ForeverYardSteps = true
	}
	for _, name := range strings.Split(os.Getenv("ARC_OFF"), ",") {
		switch strings.TrimSpace(name) {
		case "aura":
			ForeverNoAuraCritSuppression = false
		case "walkin":
			ForeverWalkInStartsAttack = false
		case "trap":
			ForeverTrapReach = false
		case "gearap":
			ForeverGearAPIsRanged = false
		case "marktsa":
			ForeverMarkAndTrueshot = false
		case "petfocus":
			ForeverPetFocusDoubled = false
		case "trapap":
			ForeverTrapAP = false
		case "sting":
			ForeverSerpentSting = false
		case "walk":
			ForeverWalkRules = false
		case "queue":
			ForeverNoSpellQueue = false
		case "seq":
			ForeverStrictSequenceNow = false
		case "ohretry":
			ForeverOffHandRetry = false
		case "wficd":
			ForeverWindfuryICD = false
		case "traparm":
			ForeverTrapArm = false
		case "traphit":
			ForeverTrapHit = false
		case "area":
			ForeverAreaReach = false
		case "gcdpoll":
			ForeverGCDPoll = false
		case "consumes":
			ForeverChosenConsumablesOnly = false
		case "ranks":
			ForeverTrainerRanks = false
		}
	}
}
