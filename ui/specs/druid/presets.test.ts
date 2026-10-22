import { readFileSync } from 'node:fs';

import { applyIndividualDefaults, type DefaultsHost } from '@features/settings/model/apply_defaults';
import { ItemSlot } from '@generated/proto/common';
import { UIDatabase } from '@generated/proto/ui';
import { brokenTalentNames } from '@sim/talents/broken_talents';
import { druidTalentsConfig } from '@sim/talents/druid';
import { parseTalentsString, totalPointsSpent } from '@sim/talents/talents_string';
import { makeIndividualSim, serveDatabase } from '@testing/sim/individual_sim';
import { beforeAll, describe, expect, it, vi } from 'vitest';

import bear from './feralbear/spec';
import cat from './feralcat/spec';

vi.mock('@sim/workers/worker_pool', async () => (await import('@testing/sim/worker_pool')).idleWorkerPool());
beforeAll(() => serveDatabase(UIDatabase.fromJsonString(readFileSync('assets/database/db.json', 'utf8'))));

describe.each([
	['Cat', cat],
	['Bear', bear],
] as const)('%s presets', (_, spec) => {
	it('offers the adaptive rotation and its weights', () => {
		expect(spec.presets.rotations).toHaveLength(1);
		expect(spec.defaults.aplRotation).toEqual(spec.presets.rotations[0].rotation.rotation);
		expect(spec.presets.epWeights).toHaveLength(1);
	});

	it('uses legal talent allocations', () => {
		for (const preset of spec.presets.talents) {
			const points = parseTalentsString(druidTalentsConfig, preset.data.talentsString);
			expect(totalPointsSpent(points)).toBe(51);
			expect(brokenTalentNames(druidTalentsConfig, points)).toEqual([]);
		}
	});

	it('retains the weightstone when applying defaults', async () => {
		const { host, player } = await makeIndividualSim({ spec: spec.spec, config: spec });
		const defaultsHost = Object.assign(host, {
			statWeightActionSettings: { applyDefaults: () => {} },
			statSweepSettings: { applyDefaults: () => {} },
		}) as unknown as DefaultsHost<typeof spec.spec>;
		applyIndividualDefaults(defaultsHost);
		expect(player.itemSwapSettings.getEnableItemSwap()).toBe(true);
		expect(player.itemSwapSettings.getGear().getEquippedItem(ItemSlot.ItemSlotMainHand)?.item.id).toBe(9449);
		expect(player.getConsumes().swapMhImbueId).toBe(12643);
	});

	it('starts with an MCP swap and the selected encounter', () => {
		expect(spec.defaults.itemSwap?.items[ItemSlot.ItemSlotMainHand].id).toBe(9449);
		expect(spec.defaults.consumables.swapMhImbueId).toBe(12643);
		expect(spec.defaults.encounter?.encounter?.duration).toBe(180);
		expect(spec.defaults.encounter?.encounter?.durationVariation).toBe(5);
		expect(spec.defaults.other?.distanceFromTarget).toBe(25);
	});
});
