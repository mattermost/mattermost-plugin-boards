{{if .postgres}}
UPDATE {{.prefix}}blocks
   SET fields = (fields::jsonb - 'attachmentId' || jsonb_build_object('fileId', fields::jsonb->'attachmentId'))::json
 WHERE jsonb_typeof(fields::jsonb->'attachmentId') = 'string'
   AND (fields::jsonb->>'attachmentId') <> '';

UPDATE {{.prefix}}blocks
   SET fields = (fields::jsonb - 'attachmentId')::json
 WHERE fields::jsonb ? 'attachmentId';

UPDATE {{.prefix}}blocks_history
   SET fields = (fields::jsonb - 'attachmentId' || jsonb_build_object('fileId', fields::jsonb->'attachmentId'))::json
 WHERE jsonb_typeof(fields::jsonb->'attachmentId') = 'string'
   AND (fields::jsonb->>'attachmentId') <> '';

UPDATE {{.prefix}}blocks_history
   SET fields = (fields::jsonb - 'attachmentId')::json
 WHERE fields::jsonb ? 'attachmentId';
{{end}}

{{if .mysql}}
UPDATE {{.prefix}}blocks
   SET fields = JSON_REMOVE(JSON_SET(fields, '$.fileId', JSON_EXTRACT(fields, '$.attachmentId')), '$.attachmentId')
 WHERE JSON_TYPE(JSON_EXTRACT(fields, '$.attachmentId')) = 'STRING'
   AND JSON_UNQUOTE(JSON_EXTRACT(fields, '$.attachmentId')) <> '';

UPDATE {{.prefix}}blocks
   SET fields = JSON_REMOVE(fields, '$.attachmentId')
 WHERE JSON_CONTAINS_PATH(fields, 'one', '$.attachmentId');

UPDATE {{.prefix}}blocks_history
   SET fields = JSON_REMOVE(JSON_SET(fields, '$.fileId', JSON_EXTRACT(fields, '$.attachmentId')), '$.attachmentId')
 WHERE JSON_TYPE(JSON_EXTRACT(fields, '$.attachmentId')) = 'STRING'
   AND JSON_UNQUOTE(JSON_EXTRACT(fields, '$.attachmentId')) <> '';

UPDATE {{.prefix}}blocks_history
   SET fields = JSON_REMOVE(fields, '$.attachmentId')
 WHERE JSON_CONTAINS_PATH(fields, 'one', '$.attachmentId');
{{end}}

{{if .sqlite}}
UPDATE {{.prefix}}blocks
   SET fields = json_remove(json_set(fields, '$.fileId', json_extract(fields, '$.attachmentId')), '$.attachmentId')
 WHERE json_type(fields, '$.attachmentId') = 'text'
   AND json_extract(fields, '$.attachmentId') <> '';

UPDATE {{.prefix}}blocks
   SET fields = json_remove(fields, '$.attachmentId')
 WHERE json_type(fields, '$.attachmentId') IS NOT NULL;

UPDATE {{.prefix}}blocks_history
   SET fields = json_remove(json_set(fields, '$.fileId', json_extract(fields, '$.attachmentId')), '$.attachmentId')
 WHERE json_type(fields, '$.attachmentId') = 'text'
   AND json_extract(fields, '$.attachmentId') <> '';

UPDATE {{.prefix}}blocks_history
   SET fields = json_remove(fields, '$.attachmentId')
 WHERE json_type(fields, '$.attachmentId') IS NOT NULL;
{{end}}
