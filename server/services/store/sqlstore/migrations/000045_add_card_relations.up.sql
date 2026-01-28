-- Create card_relations table for storing relationships between cards
CREATE TABLE IF NOT EXISTS {{.prefix}}card_relations (
    id VARCHAR(36) PRIMARY KEY,
    
    -- Source card (the card from which the relation is created)
    source_card_id VARCHAR(36) NOT NULL,
    
    -- Target card (the card being related to)
    target_card_id VARCHAR(36) NOT NULL,
    
    -- Relation type (blocks, is_blocked_by, relates_to, duplicates, is_duplicated_by, etc.)
    relation_type VARCHAR(50) NOT NULL,
    
    -- User who created this relation
    created_by VARCHAR(36) NOT NULL,
    
    -- Timestamps
    {{if .postgres}}create_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),{{end}}
    {{if .sqlite}}create_at DATETIME NOT NULL DEFAULT(STRFTIME('%Y-%m-%d %H:%M:%f', 'NOW')),{{end}}
    {{if .mysql}}create_at DATETIME(6) NOT NULL DEFAULT NOW(6),{{end}}
    
    create_at_millis BIGINT NOT NULL,
    
    -- Foreign key constraints with cascade delete
    {{if .postgres}}
    CONSTRAINT fk_source_card FOREIGN KEY (source_card_id) REFERENCES {{.prefix}}blocks(id) ON DELETE CASCADE,
    CONSTRAINT fk_target_card FOREIGN KEY (target_card_id) REFERENCES {{.prefix}}blocks(id) ON DELETE CASCADE,
    {{end}}
    
    {{if .mysql}}
    CONSTRAINT fk_source_card FOREIGN KEY (source_card_id) REFERENCES {{.prefix}}blocks(id) ON DELETE CASCADE,
    CONSTRAINT fk_target_card FOREIGN KEY (target_card_id) REFERENCES {{.prefix}}blocks(id) ON DELETE CASCADE,
    {{end}}
    
    -- Unique constraint to prevent duplicate relations
    CONSTRAINT unique_card_relation UNIQUE (source_card_id, target_card_id, relation_type)
) {{if .mysql}}DEFAULT CHARACTER SET utf8mb4{{end}};

-- Create indexes for performance
{{if .mysql}}
CREATE INDEX idx_card_relations_source ON {{.prefix}}card_relations(source_card_id);
CREATE INDEX idx_card_relations_target ON {{.prefix}}card_relations(target_card_id);
{{else}}
CREATE INDEX IF NOT EXISTS idx_card_relations_source ON {{.prefix}}card_relations(source_card_id);
CREATE INDEX IF NOT EXISTS idx_card_relations_target ON {{.prefix}}card_relations(target_card_id);
{{end}}

