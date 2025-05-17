// deno-lint-ignore-file no-process-global
// deno-lint-ignore triple-slash-reference
/// <reference types="npm:@types/node@^22.15.17" />

const is_deno = 'Deno' in globalThis;

/** Get environment variable */
export function get_env(key: string): string | undefined {
	return is_deno ? Deno.env.get(key) : process.env[key];
}

/** Set environment variable */
export function set_env(key: string, value: string): void {
	if (is_deno) {
		Deno.env.set(key, value);
	} else {
		process.env[key] = value;
	}
}

/** Get current directory */
export function cwd(): string {
	return is_deno ? Deno.cwd() : process.cwd();
}

/** Exit the process */
export function exit(code: number): never {
	return is_deno ? Deno.exit(code) : process.exit(code);
}

/** Get command-line arguments */
export function get_args(): string[] {
	return is_deno ? Deno.args : process.argv.slice(2);
}

/** Get stdout stream */
export function stdout(): WritableStream {
	return is_deno
		? Deno.stdout.writable
		: process.getBuiltinModule('stream').Writable.toWeb(
			process.stdout,
		) as WritableStream;
}

/** Get tcp connection streams */
export async function tcp_connect(
	opts: { hostname: string; port: number },
): Promise<
	{ readable: ReadableStream<Uint8Array>; writable: WritableStream<Uint8Array> }
> {
	if (is_deno) return await Deno.connect(opts);

	const { createConnection } = process.getBuiltinModule('node:net');
	const { Readable, Writable } = process.getBuiltinModule('node:stream');
	const conn = createConnection({
		host: opts.hostname,
		port: opts.port,
	});
	return {
		readable: Readable.toWeb(conn) as ReadableStream<Uint8Array>,
		writable: Writable.toWeb(conn) as WritableStream<Uint8Array>,
	};
}
