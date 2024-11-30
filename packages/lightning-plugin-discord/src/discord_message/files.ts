import type { attachment } from '@jersey/lightning';
import type { RawFile } from '@discordjs/rest';

export async function files_up_to_25MiB(attachments: attachment[] | undefined) {
    if (!attachments) return;

    const files: RawFile[] = [];
    const total_size = 0;

    for (const attachment of attachments) {
        if (attachment.size >= 25) continue;
        if (total_size + attachment.size >= 25) break;

        try {
            const data = new Uint8Array(
                await (await fetch(attachment.file, {
                    signal: AbortSignal.timeout(5000),
                })).arrayBuffer(),
            );

            files.push({
                name: attachment.name ?? attachment.file.split('/').pop()!,
                data,
            });
        } catch {
            continue;
        }
    }

    return files;
}
