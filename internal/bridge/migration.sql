CREATE OR REPLACE FUNCTION normalize_channel_id(plugin TEXT, channel TEXT)
RETURNS TEXT AS $$
BEGIN
	IF plugin LIKE 'bolt-%' THEN
		RETURN substring(plugin FROM 6) || '::' || channel;
	ELSEIF plugin <> '' THEN
		RETURN plugin || '::' || channel;
	ELSE
		RETURN split_part(channel, '::', 1) || '::' || split_part(channel, '::', 2);
	END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

ALTER TABLE bridge_messages
	DROP COLUMN IF EXISTS name,
	DROP COLUMN IF EXISTS channels,
	DROP COLUMN IF EXISTS settings;

ALTER TABLE bridge_messages ADD COLUMN tmp_messages JSONB DEFAULT '[]'::jsonb;

UPDATE bridge_messages AS bm
SET tmp_messages = sub.new_messages
FROM (
	SELECT
		bm.id,
		jsonb_agg(
			jsonb_build_object(
				'channel_id', normalize_channel_id(elem.m->>'plugin', elem.m->>'channel'),
				'message_ids', message_ids
			)
		) AS new_messages
	FROM bridge_messages AS bm
	CROSS JOIN LATERAL jsonb_array_elements(bm.messages::jsonb) AS elem(m)
	CROSS JOIN LATERAL (
		SELECT array_agg(message_id) AS message_ids
		FROM jsonb_array_elements_text(elem.m->'id') AS id_elem(message_id)
	) AS ids
	GROUP BY bm.id, normalize_channel_id(elem.m->>'plugin', elem.m->>'channel')
) AS sub
WHERE bm.id = sub.id;

ALTER TABLE bridge_messages DROP COLUMN messages;
ALTER TABLE bridge_messages RENAME COLUMN tmp_messages TO messages;

DELETE FROM bridge_messages
WHERE jsonb_array_length(messages) = 0;

ALTER TABLE bridge_messages ALTER COLUMN messages SET NOT NULL;

ALTER TABLE bridges DROP COLUMN IF EXISTS name;

DELETE FROM bridges
WHERE jsonb_array_length(channels) = 0;

INSERT INTO bridge_channels (bridge_id, channel_id, data, disabled)
SELECT DISTINCT ON (b.id, normalize_channel_id(ch.plugin, ch.id))
	b.id,
	normalize_channel_id(ch.plugin, ch.id) AS channel_id,
	ch.data,
	CASE
		WHEN jsonb_typeof(ch.disabled) = 'boolean'
			THEN jsonb_build_object('read', ch.disabled, 'write', ch.disabled)
		WHEN jsonb_typeof(ch.disabled) = 'object'
			THEN jsonb_build_object('read', ch.disabled->'read', 'write', ch.disabled->'write')
		ELSE jsonb_build_object('read', false, 'write', false)
	END AS disabled
FROM bridges b
CROSS JOIN jsonb_array_elements(b.channels)
CROSS JOIN LATERAL jsonb_to_record(jsonb_array_elements) AS ch(
	id TEXT,
	plugin TEXT,
	data JSONB,
	disabled JSONB
)
ON CONFLICT (bridge_id, channel_id) DO UPDATE
SET
	data = EXCLUDED.data,
	disabled = EXCLUDED.disabled
WHERE bridge_channels.data IS DISTINCT FROM EXCLUDED.data OR bridge_channels.disabled IS DISTINCT FROM EXCLUDED.disabled;

ALTER TABLE bridges DROP COLUMN IF EXISTS channels;

UPDATE bridges
SET settings = '{"allow_everyone": false}'::jsonb
WHERE settings IS NULL;

INSERT INTO lightning (prop, value) VALUES ('db_data_version', '0.8.1')
ON CONFLICT (prop) DO UPDATE SET value = EXCLUDED.value;
