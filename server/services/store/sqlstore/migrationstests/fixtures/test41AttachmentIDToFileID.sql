INSERT INTO focalboard_blocks
(id, board_id, channel_id, created_by, modified_by, type, title, fields, create_at, update_at, delete_at)
VALUES
('block-attach-only', 'board-1', '', 'user-1', 'user-1', 'attachment', 'attach only', '{"attachmentId":"7attachonly0000000000000001.txt"}', 1, 1, 0),
('block-attach-and-file', 'board-1', '', 'user-1', 'user-1', 'attachment', 'attach and file', '{"attachmentId":"7attachwins0000000000000002.txt","fileId":"7loserfile00000000000000003.txt"}', 1, 1, 0),
('block-empty-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'empty attach keep file', '{"attachmentId":"","fileId":"7keepfile000000000000000004.png"}', 1, 1, 0),
('block-null-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'null attach keep file', '{"attachmentId":null,"fileId":"7keepfile000000000000000005.png"}', 1, 1, 0),
('block-nonstring-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'nonstring attach keep file', '{"attachmentId":42,"fileId":"7keepfile000000000000000006.png"}', 1, 1, 0),
('block-empty-attach-no-file', 'board-1', '', 'user-1', 'user-1', 'attachment', 'empty attach no file', '{"attachmentId":""}', 1, 1, 0),
('block-file-only', 'board-1', '', 'user-1', 'user-1', 'image', 'file only', '{"fileId":"7fileonly000000000000000007.png"}', 1, 1, 0),
('block-no-file-fields', 'board-1', '', 'user-1', 'user-1', 'text', 'no file fields', '{"value":"hello"}', 1, 1, 0),
('block-empty-fields', 'board-1', '', 'user-1', 'user-1', 'card', 'empty fields', '{}', 1, 1, 0),
('block-move-with-siblings', 'board-1', '', 'user-1', 'user-1', 'attachment', 'move keeps siblings', '{"attachmentId":"7siblingmove00000000000008.txt","contentOrder":["a","b"],"other":1}', 1, 1, 0),
('block-object-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'object attach keep file', '{"attachmentId":{},"fileId":"7keepobj0000000000000009.png"}', 1, 1, 0),
('block-array-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'array attach keep file', '{"attachmentId":[],"fileId":"7keeparr0000000000000010.png"}', 1, 1, 0);

INSERT INTO focalboard_blocks_history
(id, board_id, channel_id, created_by, modified_by, type, title, fields, create_at, update_at, delete_at)
VALUES
('hist-attach-only', 'board-1', '', 'user-1', 'user-1', 'attachment', 'hist attach only', '{"attachmentId":"7histattach000000000000001.txt"}', 1, 1, 0),
('hist-empty-attach-with-file', 'board-1', '', 'user-1', 'user-1', 'image', 'hist empty attach keep file', '{"attachmentId":"","fileId":"7histkeep00000000000000002.png"}', 1, 1, 0),
('hist-file-only', 'board-1', '', 'user-1', 'user-1', 'image', 'hist file only', '{"fileId":"7histfile00000000000000003.png"}', 1, 1, 0);
