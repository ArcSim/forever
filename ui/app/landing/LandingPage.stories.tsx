import type { Meta, StoryObj } from '@storybook/react-vite';
import { withVersionsManifest } from '@testing/versions_manifest';

import { LandingPage } from './LandingPage';

const meta = {
	title: 'App/Landing/LandingPage',
	component: LandingPage,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	beforeEach: () => withVersionsManifest(new Error('offline')),
} satisfies Meta<typeof LandingPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
