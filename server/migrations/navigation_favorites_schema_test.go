package migrations_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The ids the user_favorites table's tests write besides newIssueDatabase's,
// as SQL literals: the owner of the favorites, another account, the account
// that last wrote them, a second workspace, a folder of the owner's in acme
// and his favorite of the project web.
const (
	favoriteOwner  = "'0199a2b4-0000-7000-8000-0000000000a1'"
	favoriteOther  = "'0199a2b4-0000-7000-8000-0000000000a2'"
	favoriteWriter = "'0199a2b4-0000-7000-8000-0000000000a3'"
	favoriteBeta   = "'0199a2b4-0000-7000-8000-0000000000a4'"
	ownerFolder    = "'0199a2b4-0000-7000-8000-0000000000a5'"
	webFavorite    = "'0199a2b4-0000-7000-8000-0000000000a6'"
)

// favoriteRow is the insertion of the undeleted favorite id of user in
// workspace, of kind, of the target target in the project project, named
// name, in the folder parent, at the sequence 65535, written by the writer;
// target, project, name and parent are SQL literals or NULL.
func favoriteRow(id, user, workspace, kind, target, project, name, parent string) string {
	return "INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence, " +
		"created_by_id, updated_by_id, created_at, updated_at) VALUES (" + id + ", " + workspace + ", " + project + ", " + user + ", '" +
		kind + "', " + target + ", " + name + ", " + parent + ", 65535, " + favoriteWriter + ", " + favoriteWriter + ", now(), now())"
}

// folderRow is the insertion of user's undeleted folder id named name in
// workspace (spec 2.3: no target, project or parent).
func folderRow(id, user, workspace, name string) string {
	return favoriteRow(id, user, workspace, "folder", "NULL", "NULL", "'"+name+"'", "NULL")
}

// entryRow is the insertion of user's undeleted favorite of kind, of a new
// target in project (NULL for a workspace's view), in the folder parent.
func entryRow(user, workspace, kind, project, parent string) string {
	return favoriteRow("gen_random_uuid()", user, workspace, kind, "gen_random_uuid()", project, "NULL", parent)
}

// inserted reports a statement that did not insert n rows.
func inserted(t *testing.T, pool *pgxpool.Pool, stmt string, n int64) {
	t.Helper()
	if tag, err := pool.Exec(context.Background(), stmt); err != nil || tag.RowsAffected() != n {
		t.Errorf("%s = %d rows, %v; want %d", stmt, tag.RowsAffected(), err, n)
	}
}

// The user_favorites table (M7 design 4.2, P3c spec 2.3): its columns, with
// no default on the instants; its indexes and foreign keys by name; the five
// kinds and no other; a folder with a name and no target, project or parent,
// any other favorite with a target and no name; a project's favorite of its
// own project, a cycle's and a module's in theirs; a non-empty folder name of
// up to 255 characters; a finite sequence. Among the undeleted rows an
// account favors a target once and names its folders of a workspace
// differently whatever the case, the keys free again once the row is soft
// deleted; another account, another workspace have their own. Each CHECK's
// counterexample breaks that CHECK alone: PostgreSQL tests a table's CHECKs
// in the order of their names and reports the first refusing one.
func TestUserFavoritesSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	if got, want := columnsOf(t, pool, "user_favorites"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid YES -", "user_id uuid NO -", "entity_type text NO -",
		"entity_identifier uuid YES -", "name character varying YES -", "parent_id uuid YES -", "sequence double precision NO -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO -",
		"updated_at timestamp with time zone NO -", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("user_favorites columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "user_favorites"), []string{
		"CREATE INDEX user_favorites_deleted_at_idx ON public.user_favorites USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX user_favorites_entity_type_entity_identifier_idx ON public.user_favorites USING btree (entity_type, entity_identifier) WHERE (deleted_at IS NULL)",
		"CREATE INDEX user_favorites_parent_id_idx ON public.user_favorites USING btree (parent_id)",
		"CREATE UNIQUE INDEX user_favorites_pkey ON public.user_favorites USING btree (id)",
		"CREATE INDEX user_favorites_project_id_idx ON public.user_favorites USING btree (project_id)",
		"CREATE UNIQUE INDEX user_favorites_user_id_entity_type_entity_identifier_key ON public.user_favorites USING btree (user_id, entity_type, entity_identifier) WHERE (deleted_at IS NULL)",
		"CREATE INDEX user_favorites_user_id_idx ON public.user_favorites USING btree (user_id)",
		"CREATE INDEX user_favorites_user_id_workspace_id_idx ON public.user_favorites USING btree (user_id, workspace_id) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX user_favorites_user_id_workspace_id_name_key ON public.user_favorites USING btree (user_id, workspace_id, lower((name)::text)) WHERE ((entity_type = 'folder'::text) AND (deleted_at IS NULL))",
		"CREATE INDEX user_favorites_workspace_id_idx ON public.user_favorites USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("user_favorites indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "user_favorites"), []string{
		"user_favorites_created_by_id_fkey users n", "user_favorites_parent_id_fkey user_favorites c", "user_favorites_project_id_fkey projects c",
		"user_favorites_updated_by_id_fkey users n", "user_favorites_user_id_fkey users c", "user_favorites_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("user_favorites foreign keys\n%q\nwant\n%q", got, want)
	}

	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+favoriteOwner+", 'carol@corp.com', 'x', 'carol'), ("+
			favoriteOther+", 'dave@corp.com', 'x', 'dave'), ("+favoriteWriter+", 'erin@corp.com', 'x', 'erin')",
		"INSERT INTO workspaces (id, name, slug) VALUES ("+favoriteBeta+", 'Beta', 'beta')",
		// The five kinds as each is shaped: a folder; a project's favorite of
		// its own project; a cycle's and a module's in their projects, the
		// cycle in the folder; a project's view and a workspace's.
		folderRow(ownerFolder, favoriteOwner, issueAcme, "Work"),
		favoriteRow(webFavorite, favoriteOwner, issueAcme, "project", issueWeb, issueWeb, "NULL", "NULL"),
		entryRow(favoriteOwner, issueAcme, "cycle", issueWeb, ownerFolder),
		entryRow(favoriteOwner, issueAcme, "module", issueOps, "NULL"),
		entryRow(favoriteOwner, issueAcme, "view", issueWeb, "NULL"),
		entryRow(favoriteOwner, issueAcme, "view", "NULL", ownerFolder))

	for _, kind := range []string{"issue", "page", "Project", "folders", ""} {
		refusedBy(t, pool, entryRow(favoriteOther, issueAcme, kind, issueWeb, "NULL"), "23514", "user_favorites_entity_type_check")
	}
	// A folder has a name and no target, project or parent (one level, spec
	// 2.5); any other favorite a target and no name.
	for _, stmt := range []string{
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "folder", "gen_random_uuid()", "NULL", "'Odd'", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "folder", "NULL", issueWeb, "'Odd'", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "folder", "NULL", "NULL", "'Odd'", ownerFolder),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "folder", "NULL", "NULL", "NULL", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "view", "NULL", "NULL", "NULL", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "cycle", "NULL", issueWeb, "NULL", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "view", "gen_random_uuid()", "NULL", "'Odd'", "NULL"),
		"UPDATE user_favorites SET entity_type='folder', project_id=NULL WHERE id=" + webFavorite,
		"UPDATE user_favorites SET name='Odd' WHERE id=" + webFavorite,
		"UPDATE user_favorites SET parent_id=" + ownerFolder + " WHERE id=" + ownerFolder,
	} {
		refusedBy(t, pool, stmt, "23514", "user_favorites_shape_check")
	}
	// A project's favorite is of its own project; a cycle's and a module's
	// are in theirs; a view's has its project or none.
	for _, stmt := range []string{
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "project", issueWeb, "NULL", "NULL", "NULL"),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "project", issueWeb, issueOps, "NULL", "NULL"),
		entryRow(favoriteOther, issueAcme, "cycle", "NULL", "NULL"),
		entryRow(favoriteOther, issueAcme, "module", "NULL", "NULL"),
		"UPDATE user_favorites SET project_id=NULL WHERE id=" + webFavorite,
		"UPDATE user_favorites SET project_id=" + issueOps + " WHERE id=" + webFavorite,
	} {
		refusedBy(t, pool, stmt, "23514", "user_favorites_project_id_check")
	}
	set := func(column, value string) string {
		return "UPDATE user_favorites SET " + column + "=" + value + " WHERE id=" + ownerFolder
	}
	refusedBy(t, pool, set("name", "''"), "23514", "user_favorites_name_check")
	// The name holds 255 characters, however many bytes each takes, and
	// not 256.
	accepted(t, pool, set("name", "'"+strings.Repeat("夹", 255)+"'"), set("name", "'Work'"))
	refusedBy(t, pool, set("name", "'"+strings.Repeat("夹", 256)+"'"), "22001", "")
	for _, value := range []string{"'NaN'", "'Infinity'", "'-Infinity'"} {
		refusedBy(t, pool, set("sequence", value), "23514", "user_favorites_sequence_check")
	}
	accepted(t, pool, set("sequence", "-1.5"), set("sequence", "0"), set("sequence", "1e308"), set("sequence", "65535"))
	for _, column := range []string{"workspace_id", "user_id", "entity_type", "sequence", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE user_favorites SET "+column+"=NULL WHERE id="+webFavorite, column)
	}
	nullRefused(t, pool, "INSERT INTO user_favorites (id, workspace_id, user_id, entity_type, name, sequence) VALUES (gen_random_uuid(), "+
		issueAcme+", "+favoriteOther+", 'folder', 'Fresh', 65535)", "created_at")

	// An account favors a target once among his undeleted favorites, in
	// whichever folder; the conflict target the insertion names (spec 2.3)
	// finds that key, and once the favorite is soft deleted the target is
	// favored anew.
	again := favoriteRow("gen_random_uuid()", favoriteOwner, issueAcme, "project", issueWeb, issueWeb, "NULL", ownerFolder)
	refusedBy(t, pool, again, "23505", "user_favorites_user_id_entity_type_entity_identifier_key")
	onConflict := again + " ON CONFLICT (user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL DO NOTHING"
	inserted(t, pool, onConflict, 0)
	accepted(t, pool, favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "project", issueWeb, issueWeb, "NULL", "NULL"),
		"UPDATE user_favorites SET deleted_at=now() WHERE id="+webFavorite)
	inserted(t, pool, onConflict, 1)
	refusedBy(t, pool, again, "23505", "user_favorites_user_id_entity_type_entity_identifier_key")
	// An account names his folders of a workspace differently whatever the
	// case among the undeleted ones; another account, another workspace,
	// and a soft-deleted folder's name are free.
	refusedBy(t, pool, folderRow("gen_random_uuid()", favoriteOwner, issueAcme, "WORK"), "23505", "user_favorites_user_id_workspace_id_name_key")
	accepted(t, pool, folderRow("gen_random_uuid()", favoriteOther, issueAcme, "Work"), folderRow("gen_random_uuid()", favoriteOwner, favoriteBeta, "Work"),
		"UPDATE user_favorites SET deleted_at=now() WHERE id="+ownerFolder, folderRow("gen_random_uuid()", favoriteOwner, issueAcme, "WORK"))
	refusedBy(t, pool, folderRow("gen_random_uuid()", favoriteOwner, issueAcme, "work"), "23505", "user_favorites_user_id_workspace_id_name_key")

	// Where the rows go: a folder's purge takes its favorites; the writer's
	// keeps every favorite without him; a project's takes its favorites, of
	// it, its cycles, modules and views, and not its workspace's; a
	// workspace's its own; an account's his.
	otherFolder := "'0199a2b4-0000-7000-8000-0000000000a7'"
	betaFolder := "'0199a2b4-0000-7000-8000-0000000000a8'"
	accepted(t, pool, "DELETE FROM user_favorites",
		folderRow(ownerFolder, favoriteOwner, issueAcme, "Work"),
		favoriteRow(webFavorite, favoriteOwner, issueAcme, "project", issueWeb, issueWeb, "NULL", ownerFolder),
		entryRow(favoriteOwner, issueAcme, "cycle", issueWeb, "NULL"),
		entryRow(favoriteOwner, issueAcme, "module", issueOps, "NULL"),
		entryRow(favoriteOwner, issueAcme, "view", issueWeb, "NULL"),
		entryRow(favoriteOwner, issueAcme, "view", "NULL", ownerFolder),
		folderRow(betaFolder, favoriteOwner, favoriteBeta, "Beta"),
		entryRow(favoriteOwner, favoriteBeta, "view", "NULL", betaFolder),
		folderRow(otherFolder, favoriteOther, issueAcme, "Work"),
		entryRow(favoriteOther, issueAcme, "view", "NULL", otherFolder),
		favoriteRow("gen_random_uuid()", favoriteOther, issueAcme, "project", issueWeb, issueWeb, "NULL", "NULL"),
		entryRow(favoriteOther, issueAcme, "module", issueOps, "NULL"))
	left := func(stmt string, want ...string) {
		t.Helper()
		accepted(t, pool, stmt)
		if got := names(t, pool, `SELECT concat_ws(' ', u.display_name, f.entity_type, f.name, w.slug, p.identifier, 'in ' || folder.name,
				CASE WHEN f.created_by_id IS NULL AND f.updated_by_id IS NULL THEN 'unwritten' END) COLLATE "C"
			FROM user_favorites AS f JOIN users AS u ON u.id = f.user_id JOIN workspaces AS w ON w.id = f.workspace_id
			LEFT JOIN projects AS p ON p.id = f.project_id LEFT JOIN user_favorites AS folder ON folder.id = f.parent_id ORDER BY 1`); !slices.Equal(got, want) {
			t.Fatalf("after %s, the favorites = %q, want %q", stmt, got, want)
		}
	}
	left("DELETE FROM user_favorites WHERE id="+otherFolder,
		"carol cycle acme WEB", "carol folder Beta beta", "carol folder Work acme", "carol module acme OPS", "carol project acme WEB in Work",
		"carol view acme WEB", "carol view acme in Work", "carol view beta in Beta", "dave module acme OPS", "dave project acme WEB")
	left("DELETE FROM users WHERE id="+favoriteWriter,
		"carol cycle acme WEB unwritten", "carol folder Beta beta unwritten", "carol folder Work acme unwritten", "carol module acme OPS unwritten",
		"carol project acme WEB in Work unwritten", "carol view acme WEB unwritten", "carol view acme in Work unwritten",
		"carol view beta in Beta unwritten", "dave module acme OPS unwritten", "dave project acme WEB unwritten")
	left("DELETE FROM projects WHERE id="+issueWeb,
		"carol folder Beta beta unwritten", "carol folder Work acme unwritten", "carol module acme OPS unwritten",
		"carol view acme in Work unwritten", "carol view beta in Beta unwritten", "dave module acme OPS unwritten")
	left("DELETE FROM workspaces WHERE id="+favoriteBeta,
		"carol folder Work acme unwritten", "carol module acme OPS unwritten", "carol view acme in Work unwritten", "dave module acme OPS unwritten")
	left("DELETE FROM users WHERE id="+favoriteOwner, "dave module acme OPS unwritten")
}
