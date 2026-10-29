import { ItemSlot } from '@generated/proto/common';
import i18n from '@i18n/config';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ActionId } from '.';

afterEach(() => vi.restoreAllMocks());

const filledName = async (actionId: ActionId, name: string) => {
	vi.spyOn(ActionId, 'getTooltipData').mockImplementation(async () => ({ name, icon: '' }) as never);
	return (await actionId.fill()).name;
};

describe('ActionId.fill', () => {
	const mainHand = i18n.t('common.action_tags.main_hand');
	const offHand = i18n.t('common.action_tags.off_hand');

	it("names a weapon hand's copy of a proc by the item slot it is tagged with", async () => {
		expect(await filledName(ActionId.fromSpellId(20007, 0, ItemSlot.ItemSlotMainHand), 'Holy Strength')).toBe(`Holy Strength (${mainHand})`);
		expect(await filledName(ActionId.fromSpellId(28093, 0, ItemSlot.ItemSlotOffHand), 'Lightning Speed')).toBe(`Lightning Speed (${offHand})`);
		expect(await filledName(ActionId.fromItemId(20749, ItemSlot.ItemSlotOffHand), 'Brilliant Wizard Oil')).toBe(`Brilliant Wizard Oil (${offHand})`);
	});

	it('names the hand of an aura read from the combat log', async () => {
		const actionId = ActionId.fromLogString(`Aura gained: {SpellID: 20007, Tag: ${ItemSlot.ItemSlotMainHand}}`);
		expect(await filledName(actionId, 'Holy Strength')).toBe(`Holy Strength (${mainHand})`);
	});

	it('names no hand for any other tag', async () => {
		expect(await filledName(ActionId.fromSpellId(1, 0, 1), 'Unnamed Spell')).toBe('Unnamed Spell (??)');
		expect(await filledName(ActionId.fromSpellId(1, 0, 2), 'Unnamed Spell')).toBe('Unnamed Spell (??)');
		expect(await filledName(ActionId.fromSpellId(1, 0, -1), 'Unnamed Spell')).toBe(`Unnamed Spell (${i18n.t('common.action_tags.external')})`);
	});
});
