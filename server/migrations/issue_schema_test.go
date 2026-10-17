package migrations_test

// issueNames are the constraints and indexes of the issue module's tables
// (M4 design 4.2–4.5, 4.11), as TestConstraintAndIndexNames reads them: the
// part of its want that the migration 00015 adds.
var issueNames = []string{
	"issues_created_by_id_fkey f n",
	"issues_dates_check c",
	"issues_name_check c",
	"issues_not_own_parent_check c",
	"issues_parent_id_fkey f c",
	"issues_parent_id_idx i",
	"issues_pkey iu",
	"issues_pkey p",
	"issues_priority_check c",
	"issues_project_id_fkey f c",
	"issues_project_id_idx i",
	"issues_project_id_sequence_id_key iu",
	"issues_project_id_sequence_id_key u",
	"issues_project_id_state_id_idx iw",
	"issues_sequence_id_check c",
	"issues_state_id_fkey f c",
	"issues_state_id_idx i",
	"issues_updated_by_id_fkey f n",
	"issues_workspace_id_fkey f c",
	"issues_workspace_id_idx i",
}
