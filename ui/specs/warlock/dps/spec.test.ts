import { APLRotation } from '@generated/proto/apl';
import { Cooldowns, Spec } from '@generated/proto/common';
import { Warlock_Rotation, WarlockRotation } from '@generated/proto/warlock';
import type { Player } from '@sim/player/player';
import { describe, expect, it } from 'vitest';

import AffWrackBuildJson from './builds/aff_wrack.build.json';
import DemoFireBrandBuildJson from './builds/demo_fire_brand.build.json';
import DemoShadowBrandBuildJson from './builds/demo_shadow_brand.build.json';
import DestroBuildJson from './builds/destro.build.json';
import SmRuinBuildJson from './builds/sm_ruin.build.json';
import spec from './spec';

// The generator reads nothing from the player.
const player = {} as Player<Spec.SpecWarlock>;

const generate = (simple: Partial<Warlock_Rotation>) =>
	APLRotation.toJson(
		spec.simpleRotation!(player, Warlock_Rotation.create({ rotation: WarlockRotation.WarlockRotationDemoFireBrand, ...simple }), Cooldowns.create()),
	);

const buildRotation = (json: { player: { rotation: unknown } }) => APLRotation.toJson(APLRotation.fromJson(json.player.rotation as any));

const spellIds = (json: unknown): Array<number> => {
	if (Array.isArray(json)) return json.flatMap(spellIds);
	if (json === null || typeof json !== 'object') return [];
	return Object.entries(json).flatMap(([key, value]) => (key === 'spellId' && typeof value === 'number' ? [value] : spellIds(value)));
};

const count = (json: unknown, id: number) => spellIds(json).filter(spellId => spellId === id).length;

describe('Warlock simple rotation', () => {
	it.each([
		[WarlockRotation.WarlockRotationNone, DemoFireBrandBuildJson],
		[WarlockRotation.WarlockRotationDemoFireBrand, DemoFireBrandBuildJson],
		[WarlockRotation.WarlockRotationDemoShadowBrand, DemoShadowBrandBuildJson],
		[WarlockRotation.WarlockRotationDestro, DestroBuildJson],
		[WarlockRotation.WarlockRotationSmRuin, SmRuinBuildJson],
		[WarlockRotation.WarlockRotationAffWrack, AffWrackBuildJson],
	])('runs the build rotation of choice %i as the build has it', (rotation, build) => {
		expect(generate({ rotation })).toEqual(buildRotation(build));
	});

	it.each([
		['useShadowBoltRank10', 11661, 25307],
		['useImmolateRank8', 11668, 25309],
		['useCorruptionRank7', 11672, 25311],
	] as const)('%s casts the book rank in place of trainer rank %i', (field, trainerId, bookId) => {
		const build = buildRotation(DemoShadowBrandBuildJson);
		const rotation = generate({ rotation: WarlockRotation.WarlockRotationDemoShadowBrand, [field]: true });
		expect(count(build, trainerId)).toBeGreaterThan(0);
		expect(count(rotation, trainerId)).toBe(0);
		expect(count(rotation, bookId)).toBe(count(build, trainerId));
		expect(spellIds(rotation).filter(id => id !== bookId)).toEqual(spellIds(build).filter(id => id !== trainerId));
	});
});
