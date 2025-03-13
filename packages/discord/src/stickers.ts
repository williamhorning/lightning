import type { APIAttachment, APIStickerItem } from 'discord-api-types';

export async function fetch_sticker_attachments(
	stickers?: APIStickerItem[],
): Promise<APIAttachment[] | undefined> {
	if (!stickers) return;

	return (await Promise.all(stickers.map(async (sticker) => {
		let type;

		if (sticker.format_type === 1) type = 'png';
		if (sticker.format_type === 2) type = 'apng';
		if (sticker.format_type === 3) type = 'lottie';
		if (sticker.format_type === 4) type = 'gif';

		const url = `https://media.discordapp.net/stickers/${sticker.id}.${type}`;

		const request = await fetch(url, { method: 'HEAD' });

		if (request.ok) {
			return {
				url,
				description: sticker.name,
				filename: `${sticker.name}.${type}`,
				size: parseInt(request.headers.get('Content-Length') ?? '0') / 1048576,
				id: sticker.id,
				proxy_url: url,
			};
		} else {
			return;
		}
	}))).filter((i) => i !== undefined);
}
