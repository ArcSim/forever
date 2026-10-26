import { VERSIONS_MANIFEST_URL } from '@sim/constants/other';

export type FooterLink = { name: string; url: string };
export type FooterCommunityGroup = { title: string; links: FooterLink[] };
export type FooterVersion = { title: string; url: string; status: 'live' | 'wip' };

/** The part of the shared versions manifest (schema version 1) the landing footer reads. */
export type FooterContent = {
	versions: FooterVersion[];
	communities: FooterCommunityGroup[];
	legal: { copyrightStartYear: number; copyrightHolder: string };
};

// Shown until the manifest loads, and kept when it fails or has a schema this page does not know
export const FALLBACK_FOOTER: FooterContent = {
	versions: [
		{ title: 'Forever', url: 'https://www.wowsims.com/forever/', status: 'live' },
		{ title: 'Mists of Pandaria', url: 'https://www.wowsims.com/mop/', status: 'live' },
		{ title: 'Cataclysm', url: 'https://www.wowsims.com/cata/', status: 'live' },
		{ title: 'Wrath of the Lich King', url: 'https://www.wowsims.com/wotlk/', status: 'live' },
		{ title: 'The Burning Crusade', url: 'https://www.wowsims.com/tbc/', status: 'live' },
		{ title: 'Classic/Vanilla', url: 'https://www.wowsims.com/classic/', status: 'live' },
		{ title: 'Season of Discovery', url: 'https://www.wowsims.com/sod/', status: 'live' },
	],
	communities: [
		{
			title: 'Forever Community Servers',
			links: [
				{ name: 'WoW Forever NA', url: 'https://discord.gg/wowforeverna' },
				{ name: 'Forever EU PVE', url: 'https://discord.gg/JAQMAGUyft' },
				{ name: 'Forever EU PVP', url: 'https://discord.gg/foreverclassic' },
			],
		},
		{
			title: 'Class Discords',
			links: [
				{ name: 'Death Knight', url: 'https://discord.gg/yGbC7dRxEs' },
				{ name: 'Druid', url: 'https://discord.gg/SMwmrBV' },
				{ name: 'Hunter', url: 'https://discord.gg/nzeYUcVH8v' },
				{ name: 'Mage', url: 'https://discord.gg/tEdQhsH' },
				{ name: 'Monk', url: 'https://discord.gg/FdkpqgPwHZ' },
				{ name: 'Paladin', url: 'https://discord.gg/lightclubclassic' },
				{ name: 'Priest', url: 'https://discord.gg/priestclassic' },
				{ name: 'Rogue', url: 'https://discord.gg/ADaqUftwyj' },
				{ name: 'Shaman', url: 'https://discord.gg/NvbYJNpJEE' },
				{ name: 'Warlock', url: 'https://discord.gg/nAA64gvKyB' },
				{ name: 'Warrior', url: 'https://discord.gg/RbCZJtw' },
			],
		},
	],
	legal: { copyrightStartYear: 2021, copyrightHolder: 'WoWSims team' },
};

const isObject = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null;
const isString = (value: unknown): value is string => typeof value === 'string';
const isArrayOf = <T>(value: unknown, isItem: (item: unknown) => item is T): value is T[] => Array.isArray(value) && value.every(isItem);

const isLink = (value: unknown): value is FooterLink => isObject(value) && isString(value.name) && isString(value.url);
const isGroup = (value: unknown): value is FooterCommunityGroup => isObject(value) && isString(value.title) && isArrayOf(value.links, isLink);
const isVersion = (value: unknown): value is FooterVersion =>
	isObject(value) && isString(value.title) && isString(value.url) && (value.status === 'live' || value.status === 'wip');

/** The footer content in a manifest, or null when it is not schema version 1 or a field the footer reads is malformed. */
export const parseFooterContent = (manifest: unknown): FooterContent | null => {
	if (!isObject(manifest) || manifest.schemaVersion !== 1) return null;
	const { versions, communities, legal } = manifest;
	if (!isArrayOf(versions, isVersion) || !isArrayOf(communities, isGroup)) return null;
	if (!isObject(legal) || typeof legal.copyrightStartYear !== 'number' || !isString(legal.copyrightHolder)) return null;
	return {
		versions: versions.map(({ title, url, status }) => ({ title, url, status })),
		communities: communities.map(({ title, links }) => ({ title, links: links.map(({ name, url }) => ({ name, url })) })),
		legal: { copyrightStartYear: legal.copyrightStartYear, copyrightHolder: legal.copyrightHolder },
	};
};

/** Fetches the shared manifest; resolves to null when it cannot be read. */
export const fetchFooterContent = (signal?: AbortSignal): Promise<FooterContent | null> =>
	fetch(VERSIONS_MANIFEST_URL, { signal })
		.then(response => (response.ok ? response.json() : null))
		.then(parseFooterContent)
		.catch(() => null);
