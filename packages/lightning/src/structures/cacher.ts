export class cacher<k, v> {
	private map = new Map<k, { value: v; expiry: number }>();

	constructor(private ttl: number = 30000) {}

	get(key: k): v | undefined {
		const time = Temporal.Now.instant().epochMilliseconds;
		const entry = this.map.get(key);

		if (entry && entry.expiry >= time) return entry.value;
		this.map.delete(key);
		return undefined;
	}

	set(key: k, val: v, customTtl?: number): v {
		const time = Temporal.Now.instant().epochMilliseconds;
		this.map.set(key, {
			value: val,
			expiry: time + (customTtl ?? this.ttl),
		});
		return val;
	}
}
