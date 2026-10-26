import { toastManager } from '@ui-kit/Toast';

/** A toast that holds `text` for a manual copy, for when the clipboard cannot be written. */
export const showForManualCopy = (text: string) =>
	toastManager.add({
		variant: 'error',
		autohide: false,
		body: <pre className="max-h-40 overflow-auto break-all whitespace-pre-wrap select-all">{text}</pre>,
	});
