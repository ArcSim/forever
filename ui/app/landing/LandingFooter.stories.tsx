import type { Meta, StoryObj } from '@storybook/react-vite';
import { VERSIONS_MANIFEST, withVersionsManifest } from '@testing/versions_manifest';

import { LandingFooter } from './LandingFooter';

const meta = {
	title: 'App/Landing/Components/LandingFooter',
	component: LandingFooter,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	beforeEach: ({ parameters }) => withVersionsManifest(parameters.manifest ?? new Error('offline')),
} satisfies Meta<typeof LandingFooter>;

export default meta;
type Story = StoryObj<typeof meta>;

/** The built-in content, shown when wowsims.com/versions.json cannot be read. */
export const Default: Story = {};

// Content from the shared manifest, with a version that is not live yet.
export const FromManifest: Story = {
	parameters: { manifest: VERSIONS_MANIFEST },
};
