# Arc's Forever Sims engine

The simulation engine behind [Arc's Forever Sims](https://arcs-forever-sims.pages.dev), a hunter sim for WoW Forever.
It is a fork of the [wowsims Forever](https://github.com/wowsims/forever) engine (MIT, see LICENSE):
[compare with wowsims](https://github.com/wowsims/forever/compare/master...ArcSim:forever:master).

- `sim/`: the sim itself, every class (the hunter is the one tuned for Forever)
- `ui/specs/hunter/dps/apls` and `ui/specs/hunter/dps/gear_sets`: the rotations and gear sets
- `ui/specs/hunter/dps/builds`: the settings behind every build number on the site
- `assets/database`: the item database, built by `tools/database`
- `proto/`: the settings and results formats

## Check a number

```
ARC_BUILD=../../ui/specs/hunter/dps/builds/dw_auto_weave.build.json go test --tags=with_db ./sim/hunter/ -run 'TestArcSpikeSettings$' -v
```

prints the DPS for that build as the site lists it: 20000 iterations on the site's seed, 20261012 (the path is relative
to `sim/hunter`; `ARC_ITERS` sets other iterations). The build files play God Mode weaving; for Pro Player or Regular
Player set the hunter option `weavePlay` to `WeavePerfectPlay` or `WeaveNormalPlay`. A setup exported from the site
(Export, JSON) runs the same way.

## Build and test

Needs Go and protoc with protoc-gen-go. Generate the Go code from the protos once:

```
make sim/core/proto/api.pb.go
```

Then:

```
go test --tags=with_db ./sim/hunter/ -skip 'TestHunter$|TestHunterPets$|TestStriderKickSpeedRunsOutMidWalk$|TestAutoShotWindsUpWhenTheMoveEnds$|TestAutoShotWaitsAReactionWhenTheWalkEndsOnTheMinimumRange$|TestStartAttackOnTheWayInStartsTheOffHandsHalfSwing$|TestWalkingInStartsTheAttackOnArrival$|TestAutoShotTurnsTheAttackOff$'
```

`TestHunter` and `TestHunterPets` compare against stored results from before the Forever changes. The other six are
wowsims' tests of their own walks, which the Forever walk rules replace; they pass with
`ARC_OFF=walk,walkin,ohretry,queue,seq,gcdpoll`, which turns those rules off.

The WebAssembly build the site runs:

```
GOWASM=satconv,signext GOOS=js GOARCH=wasm go build -o lib.wasm ./sim/wasm/
```
