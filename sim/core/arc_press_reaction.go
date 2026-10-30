package core

import "time"

// Realistic button presses (ArcPlay.PressReaction): an ability goes out that long after it could first be pressed (the
// GCD up, off its cooldown, in range, its condition true), as a player sees it and presses. Zero is spamming the
// button. Moves keep their own reaction.

func (action *APLAction) arcIsPress() bool {
	switch action.impl.(type) {
	case *APLActionCastSpell, *APLActionCastFriendlySpell, *APLActionChannelSpell, *APLActionMultidot, *APLActionStrictMultidot,
		*APLActionMultishield, *APLActionSequence, *APLActionStrictSequence, *APLActionGroupReference:
		return true
	}
	return false
}

// The press reaction in the fight; the pre-pull's presses go at their set times.
func (unit *Unit) arcPressReaction(sim *Simulation) time.Duration {
	if sim.CurrentTime < 0 {
		return 0
	}
	return unit.ArcPlay.PressReaction
}

// Whether the reaction to a ready press has passed. It runs from the first look that finds the press ready; until it
// ends the rotation keeps the GCD free and looks again right as it does (arcReactionEnds).
func (apl *APLRotation) arcPressReacted(sim *Simulation, action *APLAction) bool {
	reaction := apl.unit.arcPressReaction(sim)
	if reaction <= 0 || !action.arcIsPress() {
		return true
	}
	if !action.arcSeenReady {
		action.arcSeenReady, action.arcReadySince = true, sim.CurrentTime
	}
	if ends := action.arcReadySince + reaction; sim.CurrentTime < ends {
		apl.arcReactionEnds = min(apl.arcReactionEnds, ends)
		return false
	}
	return true
}
