import { HandType, ItemSlot, ItemSpec, ItemSwap, ItemType, WeaponType } from '@generated/proto/common';
import { UIDatabase, UIItem } from '@generated/proto/ui';

// Two one-hand weapons, one sharp and one blunt, for the stone imbues to follow.
export const SWORD = 1;
export const MACE = 2;

const oneHand = (id: number, weaponType: WeaponType) =>
	UIItem.create({ id, type: ItemType.ItemTypeWeapon, handType: HandType.HandTypeOneHand, weaponType, scalingOptions: { 0: { ilvl: 60 } } });

export const weaponsDatabase = () => UIDatabase.create({ items: [oneHand(SWORD, WeaponType.WeaponTypeSword), oneHand(MACE, WeaponType.WeaponTypeMace)] });

// An item swap set indexed by slot, as the saved settings write it.
export const swapSet = (weapons: { mainHand?: number; offHand?: number }) =>
	ItemSwap.create({
		items: Object.assign(new Array<ItemSpec>(ItemSlot.ItemSlotOffHand + 1).fill(ItemSpec.create()), {
			[ItemSlot.ItemSlotMainHand]: ItemSpec.create({ id: weapons.mainHand }),
			[ItemSlot.ItemSlotOffHand]: ItemSpec.create({ id: weapons.offHand }),
		}),
	});
