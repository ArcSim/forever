import i18n from '@i18n/config';
import { DISCORD_URL, GITHUB_ORG_URL } from '@sim/constants/other';
import { Button } from '@ui-kit/Button';
import { Icon } from '@ui-kit/Icon';
import { useEffect, useState } from 'react';

import { FALLBACK_FOOTER, fetchFooterContent, type FooterCommunityGroup, type FooterContent } from './footer_content';
import { LandingBrand } from './LandingBrand';

// A group with more links than this lays them out in two columns
const SPLIT_GROUP_SIZE = 6;

const FooterLinkList = ({ children, split }: { children: React.ReactNode; split?: boolean }) => (
	<ul className={split ? 'm-0 grid list-none grid-cols-2 gap-x-6 gap-y-2 p-0' : 'm-0 flex list-none flex-col gap-2 p-0'}>{children}</ul>
);

const FooterGroup = ({ id, title, wide, children }: { id: string; title: string; wide?: boolean; children: React.ReactNode }) => (
	<nav aria-labelledby={id} className={wide ? 'col-span-full lg:col-span-1' : undefined}>
		<h2 id={id} className="m-0 mb-4 text-sm font-bold text-white">
			{title}
		</h2>
		{children}
	</nav>
);

const CommunityGroup = ({ group, index }: { group: FooterCommunityGroup; index: number }) => {
	const split = group.links.length > SPLIT_GROUP_SIZE;
	return (
		<FooterGroup id={`landing-footer-community-${index}`} title={group.title} wide={split}>
			<FooterLinkList split={split}>
				{group.links.map(link => (
					<li key={link.url}>
						<Button href={link.url} target="_blank" variant="unstyled" className="ui-landing-footer-link">
							{link.name}
						</Button>
					</li>
				))}
			</FooterLinkList>
		</FooterGroup>
	);
};

export const LandingFooter = () => {
	const [content, setContent] = useState<FooterContent>(FALLBACK_FOOTER);

	useEffect(() => {
		const controller = new AbortController();
		fetchFooterContent(controller.signal).then(fetched => {
			if (fetched && !controller.signal.aborted) setContent(fetched);
		});
		return () => controller.abort();
	}, []);

	const { legal } = content;

	return (
		<footer className="border-t border-solid border-white/8 bg-background/90 text-sm text-white/65">
			<div className="ui-landing-container">
				<div className="grid grid-cols-2 gap-6 py-12 lg:grid-cols-landing-footer lg:gap-12">
					<div className="col-span-full flex flex-col items-start gap-4 lg:col-span-1">
						<LandingBrand />
						<div className="flex gap-4">
							<Button
								href={DISCORD_URL}
								target="_blank"
								variant="unstyled"
								aria-label={i18n.t('landing.header.discord')}
								className="ui-landing-footer-link">
								<Icon name="discord" style="brands" size="lg" />
							</Button>
							<Button
								href={GITHUB_ORG_URL}
								target="_blank"
								variant="unstyled"
								aria-label={i18n.t('landing.header.github')}
								className="ui-landing-footer-link">
								<Icon name="github" style="brands" size="lg" />
							</Button>
						</div>
					</div>
					<FooterGroup id="landing-footer-sims" title={i18n.t('landing.footer.sims')}>
						<FooterLinkList>
							{content.versions.map(version => (
								<li key={version.url}>
									{version.status === 'wip' ? (
										<span className="text-white/45">{version.title}</span>
									) : (
										<Button href={version.url} variant="unstyled" className="ui-landing-footer-link">
											{version.title}
										</Button>
									)}
								</li>
							))}
						</FooterLinkList>
					</FooterGroup>
					{content.communities.map((group, index) => (
						<CommunityGroup key={group.title} group={group} index={index} />
					))}
				</div>
				<div className="flex flex-wrap justify-center gap-2 border-t border-solid border-white/8 py-4 text-center text-[0.8rem] text-white/55">
					<span>
						{i18n.t('landing.footer.copyright', {
							startYear: legal.copyrightStartYear,
							year: new Date().getFullYear(),
							holder: legal.copyrightHolder,
						})}
					</span>
					<span className="max-sm:hidden" aria-hidden="true">
						·
					</span>
					<span>{i18n.t('landing.footer.disclaimer')}</span>
				</div>
			</div>
		</footer>
	);
};
