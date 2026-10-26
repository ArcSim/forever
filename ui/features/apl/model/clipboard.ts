import { APLGroup, APLListItem, APLValueVariable } from '@generated/proto/apl';

import { freshUuids } from './fresh_uuids';

export interface AplClipItems {
	action: APLListItem;
	group: APLGroup;
	variable: APLValueVariable;
}

export type AplClipKind = keyof AplClipItems;
export type AplClip = { [K in AplClipKind]: { kind: K; item: AplClipItems[K] } }[AplClipKind];

const MESSAGES = { action: APLListItem, group: APLGroup, variable: APLValueVariable };

/** Clipboard text: `{ "kind": ..., "value": <the item's proto JSON> }`, so a paste can tell an action from a group or a variable. */
export const aplClipToText = ({ kind, item }: AplClip): string =>
	JSON.stringify({ kind, value: (MESSAGES[kind] as typeof APLListItem).toJson(item as APLListItem) }, null, 2);

/** The item the text holds, with new uuids, or `undefined` when the text is not a copied action, group or variable. */
export const aplClipFromText = (text: string): AplClip | undefined => {
	try {
		const { kind, value } = JSON.parse(text);
		if (!Object.prototype.hasOwnProperty.call(MESSAGES, kind)) return undefined;
		const item = (MESSAGES[kind as AplClipKind] as typeof APLListItem).fromJson(value);
		const clip = { kind, item: freshUuids(item) } as AplClip;
		switch (clip.kind) {
			// A row with no action chosen yet, as New Action makes, is still a row.
			case 'action':
				return clip.item.action ? clip : undefined;
			case 'group':
			case 'variable':
				return clip.item.name ? clip : undefined;
		}
	} catch {
		return undefined;
	}
};

/** `name` when it is free, otherwise the first free `name (2)`, `name (3)`, ... */
export const freeName = (name: string, taken: ReadonlyArray<string>): string => {
	if (!taken.includes(name)) return name;
	for (let n = 2; ; n++) if (!taken.includes(`${name} (${n})`)) return `${name} (${n})`;
};

/** The clip's item, ready to append to `list`: a group or a variable is referenced by name, so it takes the next free one. */
export const pastedItem = (clip: AplClip, list: ReadonlyArray<object>): AplClipItems[AplClipKind] => {
	switch (clip.kind) {
		case 'action':
			return clip.item;
		case 'group':
		case 'variable':
			clip.item.name = freeName(
				clip.item.name,
				list.map(other => (other as { name: string }).name),
			);
			return clip.item;
	}
};
