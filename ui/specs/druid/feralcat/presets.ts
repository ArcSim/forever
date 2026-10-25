import * as PresetUtils from '@app/preset_utils';
import * as Consumables from '@features/settings/model/consumables_auto_gen';
import { AreaType, ConsumesSpec, Profession, PseudoStat, Race, Stat } from '@generated/proto/common';
import { FeralCatDruid_Options as FeralDruidOptions } from '@generated/proto/druid';
import { SavedTalents } from '@generated/proto/ui';
import { Stats } from '@sim/proto/stats';
import { Encounter } from '@sim/raid/encounter';

import JvmesApl from './apls/jvmes.apl.json';
import Level60GearJson from './gear_sets/default.gear.json';
import PreRaidGearJson from './gear_sets/pre_raid.gear.json';

// Another raid member's bleed keeps Rend and Tear up.
export const DefaultOptions = FeralDruidOptions.create({ assumeBleedActive: true });

export const DefaultConsumables = ConsumesSpec.create({
	flaskId: Consumables.FlaskOfNaturalAggression.id,
	foodId: Consumables.SwiftstrikeSteak.id,
	conjuredId: Consumables.DemonicRune.id,
	explosiveId: Consumables.EZThroThoriumGrenade.id,
	mhImbueId: Consumables.DenseWeightstone.id,
	swapMhImbueId: Consumables.DenseWeightstone.id,
	extraIds: [Consumables.JujuMight.id, Consumables.GroundScorpokAssay.id, Consumables.GoblinSapperCharge.id],
	elixirIds: [
		Consumables.ElixirOfCunning.id,
		Consumables.ElixirOfSages.id,
		Consumables.ElixirOfThePhalanx.id,
		Consumables.GreaterMagebloodElixir.id,
		Consumables.ElixirOfTheGrizzly.id,
	],
	resourcePotId: Consumables.MajorManaPotion.id,
	offensivePotId: Consumables.MightyRagePotion.id,
});

const encounter = Encounter.defaultEncounterProto();
encounter.areaTypes = [AreaType.AreaTypeForestGrassland, AreaType.AreaTypeMountainous];
encounter.zoneId = 616;
export const DefaultEncounter = PresetUtils.makePresetEncounter('Mount Hyjal', encounter);

export const OtherDefaults = {
	distanceFromTarget: 25,
	profession1: Profession.Engineering,
	profession2: Profession.Enchanting,
	race: Race.RaceTauren,
	reactionTime: 250,
};

// Leather at 60 with no set bonus and no proc. The stand-in the rotation was tuned on.
export const Level60Gear = PresetUtils.makePresetGear('Level 60', Level60GearJson, {
	devOnly: true,
	tooltip: 'Leather at level 60 with no set bonus and no proc. Fist of Cenarius and the Howling Idol.',
});

export const PreRaidGear = PresetUtils.makePresetGear('Pre-raid', PreRaidGearJson, {
	devOnly: true,
	tooltip: 'Pre-raid leather with The Unstoppable Force, Wolfshead Helm and full enchants.',
});

// DPS weights for Dream, Cat 9/35/7 and 180s ±120s, averaged across the four races.
// Rating weights measure hit at 7% and expertise at 5.5%.
export const DefaultEp = Stats.fromMap(
	{
		[Stat.StatStrength]: 0.5699,
		[Stat.StatAgility]: 0.5815,
		[Stat.StatAttackPower]: 0.2351,
		[Stat.StatFeralAttackPower]: 0.2351,
		[Stat.StatPhysicalDamage]: 2.1248,
		[Stat.StatHitRating]: 4.8332 / 10,
		[Stat.StatExpertiseRating]: 4.57 / 10,
		[Stat.StatCritRating]: 5.9088 / 14,
		[Stat.StatHasteRating]: 4.1595 / 10,
	},
	{
		[PseudoStat.PseudoStatMainHandDps]: 2.2704,
	},
);

export const DefaultEpWeights = PresetUtils.makePresetEpWeights('Feral Cat', DefaultEp);

// Uses Thistle Tea and optional Sapper, with racial actions gated by availability.
// Uses Dash for the approach; Sapper can be used without leaving Cat Form.
export const JVMES_APL = {
	...PresetUtils.makePresetAPLRotation('Monocat, adaptive racials and Sapper', JvmesApl),
	tooltip:
		'Maintains Faerie Fire and bleeds, pools for later Berserks, and uses optional consumables and racials. Supports an MCP opener from the weapon swap slot.',
};
export const CatTalents = PresetUtils.makePresetTalents('Cat', SavedTalents.create({ talentsString: '050022-35200032121131212051-052' }));
export const HybridTalents = PresetUtils.makePresetTalents('Hybrid', SavedTalents.create({ talentsString: '050022-35230332121132212051-' }));
