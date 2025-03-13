import { type attachment, LightningError } from '@jersey/lightning';
import type { Client } from '@jersey/rvapi';

export async function upload_attachments(
	api: Client,
	attachments?: attachment[],
): Promise<string[] | undefined> {
	if (!attachments) return undefined;

	return (await Promise.all(
		attachments.map(async (attachment) => {
			try {
				return await api.media.upload_file(
					'attachments',
					await (await fetch(attachment.file)).blob(),
				);
			} catch (e) {
				new LightningError(e, {
					message: 'Failed to upload attachment',
					extra: { original: e },
				});

				return;
			}
		}),
	)).filter((i) => i !== undefined);
}
