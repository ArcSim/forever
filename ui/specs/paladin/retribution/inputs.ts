import * as InputHelpers from '@features/inputs/input_helpers';
import { Spec } from '@generated/proto/common';
import { PaladinAura, PaladinSeal } from '@generated/proto/paladin';
import i18n from '@i18n/config';
import { Player } from '@sim/player/player';
import { ActionId } from '@sim/proto/action_id';
import { getEnumValues } from '@sim/utils/collections';

type RankedSpell = { spellId: number; rank?: number };

// The highest rank of each aura and assigned seal; Concentration Aura and Seal of Justice have one rank.
export const AURA_SPELLS: Partial<Record<PaladinAura, RankedSpell>> = {
	[PaladinAura.RetributionAura]: { spellId: 10301, rank: 5 },
	[PaladinAura.DevotionAura]: { spellId: 10293, rank: 7 },
	[PaladinAura.ConcentrationAura]: { spellId: 19746 },
	[PaladinAura.FireResistanceAura]: { spellId: 19900, rank: 3 },
	[PaladinAura.FrostResistanceAura]: { spellId: 19898, rank: 3 },
	[PaladinAura.ShadowResistanceAura]: { spellId: 19896, rank: 3 },
};
export const SEAL_SPELLS: Partial<Record<PaladinSeal, RankedSpell>> = {
	[PaladinSeal.SealOfTheCrusader]: { spellId: 20308, rank: 6 },
	[PaladinSeal.SealOfLight]: { spellId: 20349, rank: 4 },
	[PaladinSeal.SealOfWisdom]: { spellId: 20357, rank: 3 },
	[PaladinSeal.SealOfJustice]: { spellId: 20164 },
};

const spellIcon = ({ spellId, rank }: RankedSpell) => ActionId.fromSpellId(spellId, rank);

// One yard outside Judgement's 10 yd range. With an assigned seal, the Twist of Light rotation's walk in
// waits for Judgement to come into range, and a wait whose condition already holds never ends, so a start
// inside that range never reaches melee.
export const MIN_PULL_RANGE = 11;

// Each option shows the icon and tooltip of the spell it casts.
export const RotationIconInputs = [
	InputHelpers.makeRotationEnumIconInput<Spec.SpecRetributionPaladin, PaladinAura>({
		fieldName: 'aura',
		label: i18n.t('rotation_tab.options.paladin.retribution.aura.label'),
		labelTooltip: i18n.t('rotation_tab.options.paladin.retribution.aura.tooltip'),
		numColumns: 3,
		values: getEnumValues<PaladinAura>(PaladinAura)
			.filter(value => value !== PaladinAura.AuraUnknown)
			.map(value => ({ value, actionId: spellIcon(AURA_SPELLS[value]!) })),
	}),
	InputHelpers.makeRotationEnumIconInput<Spec.SpecRetributionPaladin, PaladinSeal>({
		fieldName: 'assignedSeal',
		label: i18n.t('rotation_tab.options.paladin.retribution.assigned_seal.label'),
		labelTooltip: i18n.t('rotation_tab.options.paladin.retribution.assigned_seal.tooltip'),
		numColumns: 5,
		values: [
			{ value: PaladinSeal.SealNone, tooltip: i18n.t('rotation_tab.options.paladin.retribution.assigned_seal.none') },
			...getEnumValues<PaladinSeal>(PaladinSeal)
				.filter(value => value !== PaladinSeal.SealNone)
				.map(value => ({ value, actionId: spellIcon(SEAL_SPELLS[value]!) })),
		],
	}),
];

export const RotationInputs = {
	inputs: [
		InputHelpers.makeRotationNumberInput<Spec.SpecRetributionPaladin>({
			fieldName: 'pullRange',
			label: i18n.t('rotation_tab.options.paladin.retribution.pull_range.label'),
			labelTooltip: i18n.t('rotation_tab.options.paladin.retribution.pull_range.tooltip', { min: MIN_PULL_RANGE }),
			positive: true,
			setValue: (player: Player<Spec.SpecRetributionPaladin>, newValue: number) => {
				const rotation = player.getSimpleRotation();
				rotation.pullRange = Math.max(MIN_PULL_RANGE, newValue);
				player.setSimpleRotation(rotation);
			},
		}),
		InputHelpers.makeRotationNumberInput<Spec.SpecRetributionPaladin>({
			fieldName: 'delayMajorCds',
			label: i18n.t('rotation_tab.options.paladin.retribution.delay_major_cds.label'),
			labelTooltip: i18n.t('rotation_tab.options.paladin.retribution.delay_major_cds.tooltip'),
			positive: true,
		}),
	],
};
