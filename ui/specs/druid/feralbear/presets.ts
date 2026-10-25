import * as PresetUtils from '@app/preset_utils';
import * as Consumables from '@features/settings/model/consumables_auto_gen';
import { AreaType, ConsumesSpec, HealingModel, Profession, PseudoStat, Race, Stat } from '@generated/proto/common';
import { FeralBearDruid_Options as DruidOptions } from '@generated/proto/druid';
import { SavedTalents } from '@generated/proto/ui';
import { Stats } from '@sim/proto/stats';
import { Encounter } from '@sim/raid/encounter';
import { OtherDefaults as SimUIOtherDefaults } from '@sim/spec_config';

import AdaptiveApl from './apls/adaptive.apl.json';
import Level60GearJson from './gear_sets/default.gear.json';
export const ROTATION_ADAPTIVE = PresetUtils.makePresetAPLRotation('Adaptive Bear', AdaptiveApl);

export const BearTalents = PresetUtils.makePresetTalents('Bear', SavedTalents.create({ talentsString: '050022-25200302120132012051-055' }));
export const HybridTalents = PresetUtils.makePresetTalents('Hybrid', SavedTalents.create({ talentsString: '050022-35230332121132212051-' }));

export const Level60Gear = PresetUtils.makePresetGear('Level 60', Level60GearJson, {
	devOnly: true,
	tooltip: 'Leather at level 60 with no set bonus and no proc. The idol slot is empty: every idol has an effect.',
});

const encounter = Encounter.defaultEncounterProto();
encounter.areaTypes = [AreaType.AreaTypeForestGrassland, AreaType.AreaTypeMountainous];
encounter.zoneId = 616;
encounter.targets[0].minBaseDamage = 6000;
export const Level60Encounter = PresetUtils.makePresetEncounter('Mount Hyjal', encounter);

// DPS and TMI combined so Strength and Stamina have equal DPS-equivalent weight.
// Four races, 180 ±120 seconds, Adaptive Bear and continuous Stoneshield.
// Hit is measured from 7% to 8%; expertise from 5.5% to 6.5%. Rating weights are per point.
export const BalancedEp = Stats.fromMap(
	{
		[Stat.StatStrength]: 0.5446407297,
		[Stat.StatAgility]: 0.3773568957,
		[Stat.StatStamina]: 0.5446407297,
		[Stat.StatAttackPower]: 0.2475625295,
		[Stat.StatFeralAttackPower]: 0.2475625295,
		[Stat.StatHitRating]: 0.8249079311,
		[Stat.StatCritRating]: 0.394751827,
		[Stat.StatHasteRating]: 0.4982945267,
		[Stat.StatArmorPenetration]: 0.1014276205,
		[Stat.StatExpertiseRating]: 1.9067258442,
		[Stat.StatDefenseRating]: 0.5194007453,
		[Stat.StatDodgeRating]: 0.0410571292,
		[Stat.StatArmor]: 0.1025217144,
		[Stat.StatBonusArmor]: 0.1031227206,
		[Stat.StatHealth]: 0.0415802797,
		[Stat.StatPhysicalDamage]: 1.5812011522,
		[Stat.StatArmorNoMods]: 0.0216261653,
	},
	{
		[PseudoStat.PseudoStatMainHandDps]: 3.4659375565,
	},
);

export const BalancedEpWeights = PresetUtils.makePresetEpWeights('Balanced', BalancedEp);

// Another raid member's bleed keeps Rend and Tear up.
export const DefaultOptions = DruidOptions.create({
	startingRage: 0,
	assumeBleedActive: true,
});

export const DefaultConsumables = ConsumesSpec.create({
	flaskId: Consumables.FlaskOfNaturalPrecision.id,
	foodId: Consumables.DirgesKickinChimaerokChops.id,
	conjuredId: Consumables.MajorHealthstone9421.id,
	mhImbueId: Consumables.ElementalSharpeningStone.id,
	swapMhImbueId: Consumables.DenseWeightstone.id,
	extraIds: [Consumables.GoblinSapperCharge.id, Consumables.JujuMight.id, Consumables.ScrollOfStaminaIV.id, Consumables.LungJuiceCocktail.id],
	elixirIds: [
		Consumables.GiftOfArthas.id,
		Consumables.ElixirOfTheGrizzly.id,
		Consumables.ElixirOfCunning.id,
		Consumables.ElixirOfTheOwl.id,
		Consumables.ElixirOfSages.id,
		Consumables.ElixirOfThePhalanx.id,
		Consumables.ElixirOfGreaterDefense.id,
	],
	defensivePotId: Consumables.GreaterStoneshieldPotion.id,
	offensivePotId: Consumables.MightyRagePotion.id,
});

export const OtherDefaults: Partial<SimUIOtherDefaults> = {
	profession1: Profession.Engineering,
	profession2: Profession.Enchanting,
	race: Race.RaceTauren,
	distanceFromTarget: 25,
	reactionTime: 250,
	healingModel: HealingModel.create({
		hps: 2200,
		cadenceSeconds: 0.4,
		cadenceVariation: 1.2,
		absorbFrac: 0.02,
		burstWindow: 6,
		inspirationUptime: 0.25,
	}),
};
