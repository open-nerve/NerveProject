-- +goose Up
CREATE INDEX comment_reactions_deleted_at_idx ON comment_reactions(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_reactions_deleted_at_idx ON issue_reactions(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_mentions_deleted_at_idx ON issue_mentions(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_description_versions_deleted_at_idx ON issue_description_versions(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_activities_deleted_at_idx ON issue_activities(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_comments_deleted_at_idx ON issue_comments(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_links_deleted_at_idx ON issue_links(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_relations_deleted_at_idx ON issue_relations(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_assignees_deleted_at_idx ON issue_assignees(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_labels_deleted_at_idx ON issue_labels(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issue_subscribers_deleted_at_idx ON issue_subscribers(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX draft_issues_deleted_at_idx ON draft_issues(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX issues_deleted_at_idx ON issues(deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX issues_deleted_at_idx;
DROP INDEX draft_issues_deleted_at_idx;
DROP INDEX issue_subscribers_deleted_at_idx;
DROP INDEX issue_labels_deleted_at_idx;
DROP INDEX issue_assignees_deleted_at_idx;
DROP INDEX issue_relations_deleted_at_idx;
DROP INDEX issue_links_deleted_at_idx;
DROP INDEX issue_comments_deleted_at_idx;
DROP INDEX issue_activities_deleted_at_idx;
DROP INDEX issue_description_versions_deleted_at_idx;
DROP INDEX issue_mentions_deleted_at_idx;
DROP INDEX issue_reactions_deleted_at_idx;
DROP INDEX comment_reactions_deleted_at_idx;
