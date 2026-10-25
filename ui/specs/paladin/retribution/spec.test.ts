import { APLGroup, APLRotation } from '@generated/proto/apl';
import { Cooldowns, Spec } from '@generated/proto/common';
import { PaladinAura, PaladinSeal, RetributionPaladin_Rotation } from '@generated/proto/paladin';
import type { Player } from '@sim/player/player';
import { describe, expect, it } from 'vitest';

import HolyShockApl from './apls/holy_shock.apl.json';
import TwistOfLightApl from './apls/twist_of_light.apl.json';
import { DefaultSimpleRotation } from './presets';
import spec from './spec';

// The generator reads nothing from the player but the Holy Shock talent.
const player = (holyShock: boolean) => ({ getTalents: () => ({ holyShock }) }) as unknown as Player<Spec.SpecRetributionPaladin>;

const generate = (simple: Partial<RetributionPaladin_Rotation>, holyShock = false, cooldowns = Cooldowns.create()) =>
	spec.simpleRotation!(player(holyShock), RetributionPaladin_Rotation.create({ ...DefaultSimpleRotation, ...simple }), cooldowns);

const variable = (rotation: APLRotation, name: string) => {
	const value = rotation.valueVariables.find(v => v.name === name)?.value?.value;
	return value?.oneofKind === 'const' ? value.const.val : undefined;
};

const group = (rotation: APLRotation, name: string) => APLGroup.toJson(rotation.groups.find(g => g.name === name)!);

const castGroup = (name: string, spellId: object) => ({ name, actions: [{ action: { castSpell: { spellId } } }] });

const aplGroup = (apl: any, name: string) => APLGroup.toJson(APLGroup.fromJson(apl.groups.find((g: any) => g.name === name)));

describe('Retribution simple rotation', () => {
	it('builds on the Holy Shock APL with the talent and the Twist of Light APL without it', () => {
		expect(generate({}, true).groups.map(g => g.name)).toEqual(HolyShockApl.groups.map(g => g.name));
		expect(generate({}, false).groups.map(g => g.name)).toEqual(TwistOfLightApl.groups.map(g => g.name));
	});

	it.each([
		[PaladinSeal.SealOfTheCrusader, { spellId: 20308, rank: 6 }],
		[PaladinSeal.SealOfLight, { spellId: 20349, rank: 4 }],
		[PaladinSeal.SealOfWisdom, { spellId: 20357, rank: 3 }],
		[PaladinSeal.SealOfJustice, { spellId: 20164 }],
	])('casts assigned seal %i in Cast assigned seal', (assignedSeal, spellId) => {
		const rotation = generate({ assignedSeal });
		expect(variable(rotation, 'Has assigned seal')).toBe('true');
		expect(group(rotation, 'Cast assigned seal')).toEqual(castGroup('Cast assigned seal', spellId));
	});

	it('leaves Cast assigned seal as the APL has it when no seal is assigned', () => {
		const rotation = generate({ assignedSeal: PaladinSeal.SealNone });
		expect(variable(rotation, 'Has assigned seal')).toBe('false');
		expect(group(rotation, 'Cast assigned seal')).toEqual(aplGroup(TwistOfLightApl, 'Cast assigned seal'));
	});

	it('leaves Activate assigned aura as the APL has it when no aura is set', () => {
		expect(group(generate({ aura: PaladinAura.AuraUnknown }), 'Activate assigned aura')).toEqual(aplGroup(TwistOfLightApl, 'Activate assigned aura'));
	});

	it.each([
		[PaladinAura.RetributionAura, { spellId: 10301, rank: 5 }],
		[PaladinAura.DevotionAura, { spellId: 10293, rank: 7 }],
		[PaladinAura.ConcentrationAura, { spellId: 19746 }],
		[PaladinAura.FireResistanceAura, { spellId: 19900, rank: 3 }],
		[PaladinAura.FrostResistanceAura, { spellId: 19898, rank: 3 }],
		[PaladinAura.ShadowResistanceAura, { spellId: 19896, rank: 3 }],
	])('casts aura %i in Activate assigned aura', (aura, spellId) => {
		expect(group(generate({ aura }), 'Activate assigned aura')).toEqual(castGroup('Activate assigned aura', spellId));
	});

	it.each([
		[0, '25'],
		[6, '11'],
		[10, '11'],
		[11, '11'],
		[30, '30'],
	])('sets Pull range from a starting range of %i to %s', (pullRange, expected) => {
		expect(variable(generate({ pullRange }), 'Pull range')).toBe(expected);
	});

	it('sets Delay Major CDs in seconds', () => {
		expect(variable(generate({ delayMajorCds: 11 }), 'Delay Major CDs')).toBe('11s');
	});

	it('puts the scheduled cooldowns ahead of the priority list', () => {
		const cooldowns = Cooldowns.create({ cooldowns: [{ id: { rawId: { oneofKind: 'spellId', spellId: 1310905 } }, timings: [10] }] });
		const rotation = generate({}, false, cooldowns);
		expect(rotation.priorityList).toHaveLength(TwistOfLightApl.priorityList.length + 1);
		expect(rotation.priorityList[0].action?.action.oneofKind).toBe('schedule');
	});
});
