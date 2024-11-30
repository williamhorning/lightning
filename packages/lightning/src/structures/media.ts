/** attachments within a message */
export interface attachment {
	/** alt text for images */
	alt?: string;
	/** a URL pointing to the file */
	file: string;
	/** the file's name */
	name?: string;
	/** whether or not the file has a spoiler */
	spoiler?: boolean;
	/** file size in MiB */
	size: number;
}

/** a discord-style embed */
export interface embed {
	/** the author of the embed */
	author?: {
		/** the name of the author */
		name: string;
		/** the url of the author */
		url?: string;
		/** the icon of the author */
		icon_url?: string;
	};
	/** the color of the embed */
	color?: number;
	/** the text in an embed */
	description?: string;
	/** fields within the embed */
	fields?: {
		/** the name of the field */
		name: string;
		/** the value of the field */
		value: string;
		/** whether or not the field is inline */
		inline?: boolean;
	}[];
	/** a footer shown in the embed */
	footer?: {
		/** the footer text */
		text: string;
		/** the icon of the footer */
		icon_url?: string;
	};
	/** an image shown in the embed */
	image?: media;
	/** a thumbnail shown in the embed */
	thumbnail?: media;
	/** the time (in epoch ms) shown in the embed */
	timestamp?: number;
	/** the title of the embed */
	title?: string;
	/** a site linked to by the embed */
	url?: string;
	/** a video inside of the embed */
	video?: media;
}

/** media inside of an embed */
export interface media {
	/** the height of the media */
	height?: number;
	/** the url of the media */
	url: string;
	/** the width of the media */
	width?: number;
}
