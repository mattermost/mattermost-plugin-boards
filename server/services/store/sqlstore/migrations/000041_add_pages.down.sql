{{- /* Reverse pages migration */ -}}

DROP TABLE IF EXISTS {{.prefix}}page_links;
DROP TABLE IF EXISTS {{.prefix}}page_acl;
DROP TABLE IF EXISTS {{.prefix}}page_channels;
DROP TABLE IF EXISTS {{.prefix}}page_yjs_updates;
DROP TABLE IF EXISTS {{.prefix}}page_content;

{{ dropColumnIfNeeded "boards_history" "layout" }}
{{ dropColumnIfNeeded "boards" "layout" }}
