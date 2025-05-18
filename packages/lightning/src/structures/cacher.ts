/** a class that wraps map to cache keys */
export class cacher<k, v> {
	/** the map used to internally store keys */
	private map = new Map<k, { value: v; expiry: number }>();

	/** create a cacher with a ttl (defaults to 30000) */
	constructor(private ttl: number = 30000) {}

	/** get a key from the map, returning undefined if expired or not found */
	get(key: k): v | undefined {
		const time = Temporal.Now.instant().epochMilliseconds;
		const entry = this.map.get(key);

		if (entry && entry.expiry >= time) return entry.value;
		this.map.delete(key);
		return undefined;
	}

	/** set a key in the map along with its expiry */
	set(key: k, val: v, customTtl?: number): v {
		const time = Temporal.Now.instant().epochMilliseconds;
		this.map.set(key, {
			value: val,
			expiry: time + (customTtl ?? this.ttl),
		});
		return val;
	}
}
