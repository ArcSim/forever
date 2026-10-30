package hunter

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// How far a shot or sting cast in an Auto Shot cycle pushes that cycle's Auto Shot back (logs, standing still: +0.19 s
// middle with one cast in the cycle, +0.00 s with none).
const shotAutoShotDelay = 200 * time.Millisecond

// The move and press reactions, the weaving mode and the run speed.
func (hunter *Hunter) applyPlayOptions(options *proto.HunterOptions) {
	play := &hunter.ArcPlay
	if options.MoveDelayMs != nil {
		play.MoveDelay, play.MoveDelaySet = time.Duration(options.GetMoveDelayMs())*time.Millisecond, true
	}
	// Pro Player and Regular Player: a weave of 2.5-2.6 s or 2.8-3.0 s with a 6.67 yd dead zone, from players'
	// measured weaves
	play.WeavePlay = int32(options.WeavePlay)
	switch options.WeavePlay {
	case proto.HunterOptions_WeavePerfectPlay:
		play.WeaveReactionMin, play.WeaveReactionSpan, play.WeaveOvershoot = 100*time.Millisecond, 100*time.Millisecond, 2.0/3
	case proto.HunterOptions_WeaveNormalPlay:
		play.WeaveReactionMin, play.WeaveReactionSpan, play.WeaveOvershoot = 200*time.Millisecond, 100*time.Millisecond, 2.0/3
	}
	play.BaseRunSpeed = options.RunSpeed
	play.PressReaction = time.Duration(options.PressReactionMs) * time.Millisecond

	if options.ShotsDelayAutoShot {
		// a new Auto Shot cycle: a shot in it may push the next Auto Shot back again
		rangedConfig := hunter.AutoAttacks.RangedConfig()
		applyEffects := rangedConfig.ApplyEffects
		rangedConfig.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			hunter.autoShotDelayed = false
			applyEffects(sim, target, spell)
		}
	}
}

// Shots delay Auto Shot: the first shot or sting cast in an Auto Shot cycle pushes its Auto Shot back.
func (hunter *Hunter) applyShotsDelayAutoShot() {
	if !hunter.Options.ShotsDelayAutoShot {
		return
	}
	core.MakePermanent(hunter.RegisterAura(core.Aura{
		Label: "Shots Delay Auto Shot",
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if hunter.autoShotDelayed || !spell.ProcMask.Matches(core.ProcMaskRangedSpecial) {
				return
			}
			hunter.autoShotDelayed = true
			hunter.AutoAttacks.DelayRangedBy(sim, shotAutoShotDelay)
		},
	}))
}
