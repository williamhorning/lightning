import type { attachment } from '@jersey/lightning';
import type { Client } from 'guilded.js';

export async function fetch_attachments(
	bot: Client,
	urls: string[],
): Promise<attachment[]> {
	const attachments: attachment[] = [];

	try {
		const signed =
			await (await fetch('https://www.guilded.gg/api/v1/url-signatures', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'application/json',
					'Authorization': `Bearer ${bot.token}`,
				},
				body: JSON.stringify({
					urls: urls.map((url) => (url.split('(').pop()?.split(')')[0])),
				}),
			})).json();

		for (const url of signed.urlSignatures || []) {
			if (url.signature) {
				const resp = await fetch(url.signature, {
					method: 'HEAD',
				});

				attachments.push({
					name: url.signature.split('/').pop()?.split('?')[0] || 'unknown',
					file: url.signature,
					size: parseInt(resp.headers.get('Content-Length') || '0') / 1048576,
				});
			}
		}
	} catch {
		// ignore
	}

	return attachments;
}
