/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueSearch } from "@/hooks/use-issue-search";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { useSessionSWR } from "@/lib/use-session-swr";
import type { TPowerKContext, TPowerKPageType } from "../../core/types";
import { PowerKModalNoSearchResultsCommand } from "./no-results-command";
import { PowerKModalSearchResults } from "./search-results";

type Props = {
  activePage: TPowerKPageType | null;
  context: TPowerKContext;
  isWorkspaceLevel: boolean;
  searchTerm: string;
  updateSearchTerm: (value: string) => void;
  isOpen: boolean;
};

export const PowerKModalSearchMenu = observer(function PowerKModalSearchMenu(props: Props) {
  const { activePage, context, isWorkspaceLevel, searchTerm, updateSearchTerm, isOpen } = props;
  const { rootIssueStore: root } = useIssueDetail();
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  const { workspaceSlug, projectId } = context.params;
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug ?? "");
  const directory = root.rootStore.projectRoot.project;
  const open = root.active && isOpen && !!workspace;
  const searching = open && !activePage && searchTerm.trim().length > 0;
  const catalog = useSessionSWR(open && workspace ? ["PROJECTS", workspace.id, workspace.slug] : null, (id, slug) =>
    directory.fetchProjects({ id, slug })
  );
  const search = useIssueSearch(
    root,
    workspaceSlug,
    { q: searchTerm, ...(!isWorkspaceLevel && projectId ? { project_id: projectId } : {}) },
    searching
  );
  if (!searching || !workspace) return null;

  const needle = searchTerm.toLowerCase();
  const projects = directory.knownProjectIds(workspace.id).flatMap((id) => {
    const project = directory.getProjectById(id);
    return project &&
      project.workspace_id === workspace.id &&
      !project.archived_at &&
      (isWorkspaceLevel || !projectId || id === projectId) &&
      (project.name.toLowerCase().includes(needle) || project.identifier.toLowerCase().includes(needle))
      ? [project]
      : [];
  });
  const issues = search.ids.flatMap((id) => {
    const issue = root.issues.getIssueById(id);
    const project = issue ? directory.getProjectById(issue.project_id) : undefined;
    return issue && project && project.workspace_id === workspace.id ? [{ issue, project }] : [];
  });
  const incomplete = issues.length !== search.ids.length;
  const error =
    search.error ?? catalog.error ?? (incomplete ? new Error("Search directory is unavailable") : undefined);
  if (error)
    return (
      <div role="alert" className="p-4 text-13 text-secondary">
        <p>{refusalMessage(error)}</p>
        <button
          className="mt-2 underline"
          onClick={() => {
            if (search.error || incomplete) search.retry();
            if (catalog.error) void catalog.mutate().catch(() => undefined);
          }}
        >
          {t("common.retry")}
        </button>
      </div>
    );
  if (search.loading || catalog.data === undefined)
    return (
      <div role="status" className="p-4 text-13 text-secondary">
        {t("common.loading")}
      </div>
    );

  return (
    <>
      {issues.length === 0 && projects.length === 0 && (
        <PowerKModalNoSearchResultsCommand updateSearchTerm={updateSearchTerm} />
      )}
      <PowerKModalSearchResults
        workspaceSlug={workspace.slug}
        issues={issues}
        projects={projects}
        closePalette={context.closePalette}
      />
    </>
  );
});
