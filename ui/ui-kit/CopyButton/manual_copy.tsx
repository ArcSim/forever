import { toastManager } from '@ui-kit/Toast';

/** A toast that holds `text` for a manual copy, for when the clipboard cannot be written. */
export const showForManualCopy = (text: string) =>
	toastManager.add({
		variant: 'error',
		autohide: false,
		body: <pre className="max-h-40 overflow-auto break-all whitespace-pre-wrap select-all">{text}</pre>,
	});

/**
 * Writes `text` to the clipboard. With no `navigator.clipboard` (an insecure origin) or a refused write, a toast
 * holds the text for a manual copy. Resolves whether the write landed.
 */
export const writeToClipboard = (text: string): Promise<boolean> =>
	(navigator.clipboard?.writeText(text) ?? Promise.reject()).then(
		() => true,
		() => {
			showForManualCopy(text);
			return false;
		},
	);
