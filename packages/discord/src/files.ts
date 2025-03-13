import type { RawFile } from '@discordjs/rest';
import type { attachment } from '@jersey/lightning';

export async function fetch_files(
	attachments: attachment[] | undefined,
): Promise<RawFile[] | undefined> {
	if (!attachments) return;

	let total_size = 0;

	return (await Promise.all(
		attachments.map(async (attachment) => {
			try {
				if (attachment.size >= 25) return;
				if (total_size + attachment.size >= 25) return;

				const data = new Uint8Array(
					await (await fetch(attachment.file, {
						signal: AbortSignal.timeout(5000),
					})).arrayBuffer(),
				);

				const name = attachment.name ?? attachment.file?.split('/').pop()!;

				total_size += attachment.size;

				return { data, name };
			} catch {
				return;
			}
		}),
	)).filter((i) => i !== undefined);
}
