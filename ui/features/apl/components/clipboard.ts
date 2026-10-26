import { type AplClip, aplClipFromText, type AplClipItems, type AplClipKind, aplClipToText, pastedItem } from '@features/apl/model/clipboard';
import i18n from '@i18n/config';
import { showForManualCopy } from '@ui-kit/CopyButton';
import { toastManager } from '@ui-kit/Toast';

const LABELS: Record<AplClipKind, { item: string; list: string }> = {
	action: { item: 'rotation_tab.apl.priorityList.name', list: 'rotation_tab.apl.tabs.priorityList' },
	group: { item: 'rotation_tab.apl.actionGroups.name', list: 'rotation_tab.apl.tabs.actionGroups' },
	variable: { item: 'rotation_tab.apl.variables.name', list: 'rotation_tab.apl.tabs.variables' },
};

/**
 * Writes the item to the clipboard and toasts. With no `navigator.clipboard` (an insecure origin) or a refused
 * write, the toast shows the text for a manual copy.
 */
export const copyAplClip = (clip: AplClip) => {
	if (!clip.item) return;
	const text = aplClipToText(clip);
	if (navigator.clipboard == undefined) {
		showForManualCopy(text);
		return;
	}
	navigator.clipboard.writeText(text).then(
		() => toastManager.add({ variant: 'success', body: i18n.t('rotation_tab.apl.clipboard.copied', { itemName: i18n.t(LABELS[clip.kind].item) }) }),
		() => showForManualCopy(text),
	);
};

/** The clipboard's item when it is a `kind`; otherwise a toast says why nothing pastes. */
const readAplClip = async (kind: AplClipKind): Promise<AplClip | undefined> => {
	const text = await navigator.clipboard?.readText().catch(() => undefined);
	if (text === undefined) {
		toastManager.add({ variant: 'error', body: i18n.t('rotation_tab.apl.clipboard.unreadable') });
		return undefined;
	}
	const clip = aplClipFromText(text);
	if (!clip) {
		toastManager.add({ variant: 'error', body: i18n.t('rotation_tab.apl.clipboard.empty', { itemName: i18n.t(LABELS[kind].item) }) });
		return undefined;
	}
	if (clip.kind !== kind) {
		toastManager.add({
			variant: 'warning',
			body: i18n.t('rotation_tab.apl.clipboard.wrong_kind', { held: i18n.t(LABELS[clip.kind].item), heldList: i18n.t(LABELS[clip.kind].list) }),
		});
		return undefined;
	}
	return clip;
};

/** The `useListItems` binding of the list a paste appends to. */
export interface AplPasteTarget<T> {
	items: ReadonlyArray<T>;
	append: (item: T) => void;
}

/** Appends the clipboard's item to the end of the list, under the next free name when it is a group or a variable whose name is taken. */
export const pasteAplClip = <K extends AplClipKind>(kind: K, list: AplPasteTarget<AplClipItems[K]>) =>
	void readAplClip(kind).then(clip => {
		if (clip) list.append(pastedItem(clip, list.items) as AplClipItems[K]);
	});
