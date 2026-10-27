{{if .postgres}}
UPDATE {{.prefix}}blocks
   SET fields = (CASE
         WHEN jsonb_typeof(fields::jsonb->'attachmentId') = 'string'
          AND (fields::jsonb->>'attachmentId') <> ''
         THEN fields::jsonb - 'attachmentId' || jsonb_build_object('fileId', fields::jsonb->'attachmentId')
         ELSE fields::jsonb - 'attachmentId'
       END)::json
 WHERE fields::text LIKE '%"attachmentId"%'
   AND fields::jsonb ? 'attachmentId';

UPDATE {{.prefix}}blocks_history
   SET fields = (CASE
         WHEN jsonb_typeof(fields::jsonb->'attachmentId') = 'string'
          AND (fields::jsonb->>'attachmentId') <> ''
         THEN fields::jsonb - 'attachmentId' || jsonb_build_object('fileId', fields::jsonb->'attachmentId')
         ELSE fields::jsonb - 'attachmentId'
       END)::json
 WHERE fields::text LIKE '%"attachmentId"%'
   AND fields::jsonb ? 'attachmentId';
{{end}}

{{if .mysql}}
UPDATE {{.prefix}}blocks
   SET fields = (CASE
         WHEN JSON_TYPE(JSON_EXTRACT(fields, '$.attachmentId')) = 'STRING'
          AND JSON_UNQUOTE(JSON_EXTRACT(fields, '$.attachmentId')) <> ''
         THEN JSON_REMOVE(JSON_SET(fields, '$.fileId', JSON_EXTRACT(fields, '$.attachmentId')), '$.attachmentId')
         ELSE JSON_REMOVE(fields, '$.attachmentId')
       END)
 WHERE fields LIKE '%"attachmentId"%'
   AND JSON_CONTAINS_PATH(fields, 'one', '$.attachmentId');

UPDATE {{.prefix}}blocks_history
   SET fields = (CASE
         WHEN JSON_TYPE(JSON_EXTRACT(fields, '$.attachmentId')) = 'STRING'
          AND JSON_UNQUOTE(JSON_EXTRACT(fields, '$.attachmentId')) <> ''
         THEN JSON_REMOVE(JSON_SET(fields, '$.fileId', JSON_EXTRACT(fields, '$.attachmentId')), '$.attachmentId')
         ELSE JSON_REMOVE(fields, '$.attachmentId')
       END)
 WHERE fields LIKE '%"attachmentId"%'
   AND JSON_CONTAINS_PATH(fields, 'one', '$.attachmentId');
{{end}}

{{if .sqlite}}
UPDATE {{.prefix}}blocks
   SET fields = (CASE
         WHEN json_type(fields, '$.attachmentId') = 'text'
          AND json_extract(fields, '$.attachmentId') <> ''
         THEN json_remove(json_set(fields, '$.fileId', json_extract(fields, '$.attachmentId')), '$.attachmentId')
         ELSE json_remove(fields, '$.attachmentId')
       END)
 WHERE fields LIKE '%"attachmentId"%'
   AND json_type(fields, '$.attachmentId') IS NOT NULL;

UPDATE {{.prefix}}blocks_history
   SET fields = (CASE
         WHEN json_type(fields, '$.attachmentId') = 'text'
          AND json_extract(fields, '$.attachmentId') <> ''
         THEN json_remove(json_set(fields, '$.fileId', json_extract(fields, '$.attachmentId')), '$.attachmentId')
         ELSE json_remove(fields, '$.attachmentId')
       END)
 WHERE fields LIKE '%"attachmentId"%'
   AND json_type(fields, '$.attachmentId') IS NOT NULL;
{{end}}
