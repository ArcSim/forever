import i18n from '@i18n/config';
import { ADDON_URL, DISCORD_URL, GITHUB_ORG_URL, PATREON_URL } from '@sim/constants/other';
import { VERSIONS_MANIFEST, withVersionsManifest } from '@testing/versions_manifest';
import { render, screen } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import english from '../../../assets/locales/en/translation.json';
import { LandingFooter } from './LandingFooter';

beforeAll(() => {
	i18n.addResourceBundle('en', 'translation', english, true, true);
});
afterAll(() => {
	i18n.removeResourceBundle('en', 'translation');
});

let restoreFetch = () => {};
afterEach(() => restoreFetch());

const year = new Date().getFullYear();

describe('LandingFooter', () => {
	it('shows the built-in content when the manifest cannot be fetched', async () => {
		restoreFetch = withVersionsManifest(new Error('offline'));
		render(<LandingFooter />);
		expect(await screen.findByRole('link', { name: 'Death Knight' })).toBeTruthy();
		expect(screen.getByRole('contentinfo').textContent).toContain(`© 2021-${year} WoWSims team`);
	});

	it('keeps the built-in content for a manifest schema it does not know', async () => {
		restoreFetch = withVersionsManifest({ ...VERSIONS_MANIFEST, schemaVersion: 2 });
		render(<LandingFooter />);
		await new Promise(resolve => setTimeout(resolve, 0));
		expect(screen.queryByRole('link', { name: 'Example Realm' })).toBeNull();
		expect(screen.getByRole('link', { name: 'Forever' })).toBeTruthy();
	});

	it('shows the versions, communities and legal line from the manifest', async () => {
		restoreFetch = withVersionsManifest(VERSIONS_MANIFEST);
		render(<LandingFooter />);
		const realm = await screen.findByRole('link', { name: 'Example Realm' });
		expect(realm.getAttribute('href')).toBe('https://discord.gg/example');
		expect(realm.getAttribute('target')).toBe('_blank');
		expect(screen.getByRole('navigation', { name: 'Partner Servers' })).toBeTruthy();
		expect(screen.queryByRole('link', { name: 'Death Knight' })).toBeNull();
		expect(screen.getByRole('contentinfo').textContent).toContain(`© 2020-${year} WoWSims contributors`);
	});

	it('links live versions in the same tab and leaves a wip version unlinked', async () => {
		restoreFetch = withVersionsManifest(VERSIONS_MANIFEST);
		render(<LandingFooter />);
		await screen.findByRole('link', { name: 'Example Realm' });
		const mop = screen.getByRole('link', { name: 'Mists of Pandaria' });
		expect(mop.getAttribute('href')).toBe('https://www.wowsims.com/mop/');
		expect(mop.getAttribute('target')).toBeNull();
		expect(screen.getByText('Next Expansion').closest('a')).toBeNull();
	});

	it('links Discord, GitHub, the addon and Patreon in new tabs', () => {
		restoreFetch = withVersionsManifest(new Error('offline'));
		render(<LandingFooter />);
		const links: [string, string][] = [
			['landing.header.discord', DISCORD_URL],
			['landing.header.github', GITHUB_ORG_URL],
			['landing.header.addon', ADDON_URL],
			['landing.header.patreon', PATREON_URL],
		];
		links.forEach(([label, url]) => {
			const link = screen.getByRole('link', { name: i18n.t(label) });
			expect(link.getAttribute('href')).toBe(url);
			expect(link.getAttribute('target')).toBe('_blank');
		});
	});
});
