import type { message } from '@jersey/lightning';
import convert_markdown from 'telegramify-markdown';

export function get_outgoing(
	msg: message,
	bridged: boolean,
): { function: 'sendMessage' | 'sendDocument'; value: string }[] {
	let content = bridged
		? `${msg.author.username} » ${msg.content || '_no content_'}`
		: msg.content ?? '_no content_';

	if ((msg.embeds?.length ?? 0) > 0) {
		content += '\n_this message has embeds_';
	}

	const messages: {
		function: 'sendMessage' | 'sendDocument';
		value: string;
	}[] = [{
		function: 'sendMessage',
		value: convert_markdown(content, 'escape'),
	}];

	for (const attachment of (msg.attachments ?? [])) {
		messages.push({
			function: 'sendDocument',
			value: attachment.file,
		});
	}

	return messages;
}
