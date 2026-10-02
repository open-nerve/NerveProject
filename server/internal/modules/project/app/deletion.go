package app

import "context"

// deleteProjects runs the steps of a deletion of projects, in the global
// order of M3 design 3.6: the projects, their memberships, the members'
// display settings, the states; P7 adds the labels at the end. Deleting a
// workspace (Cascade) and deleting a project both run it, so neither can
// leave out a table the other deletes. A failing step comes back as itself
// and the steps after it do not run.
func deleteProjects(ctx context.Context, p ProjectsDeleter, d Deletion) error {
	for _, step := range []func(context.Context, Deletion) error{
		p.DeleteProjects, p.DeleteProjectMembers, p.DeleteProjectPreferences, p.DeleteStates,
	} {
		if err := step(ctx, d); err != nil {
			return err
		}
	}
	return nil
}
