/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLayoutEffect, useMemo, useState } from "react";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueSearchQuery } from "@/services/issue/issue-search.service";
import { DetailResource } from "@/store/issue/issue-details/resource";
import type { IssueReadToken, IssueRootStore } from "@/store/issue/root.store";

/** Query lifetime only: entities and permission generations belong to the session's issue root. */
export function useIssueSearch(root: IssueRootStore, slug: string | undefined, query: IssueSearchQuery, open: boolean) {
  const workspaceID = slug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(slug)?.id : undefined;
  const projectIDs = query.project_id ? [query.project_id] : [];
  const permission = root.permissionReads.identity(workspaceID, projectIDs);
  const allowed =
    !query.project_id ||
    (!!slug &&
      root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(slug, query.project_id) !== undefined);
  const active = root.active;
  const serialized = JSON.stringify(query);
  const [attempt, setAttempt] = useState(0);
  const owner = useMemo(
    () => ({
      root,
      slug,
      workspaceID,
      query: JSON.parse(serialized) as IssueSearchQuery,
      permission,
      attempt,
      enabled: open && active && allowed && !!slug && !!workspaceID,
    }),
    [root, slug, workspaceID, serialized, open, active, permission, allowed, attempt]
  );
  const [held, setHeld] = useState<{
    owner: typeof owner;
    resource: DetailResource<readonly string[]>;
    proof?: IssueReadToken;
  }>();

  useLayoutEffect(() => {
    if (!owner.enabled || !owner.workspaceID || !owner.slug) return;
    const { root: issueRoot, slug: workspaceSlug, workspaceID: workspace, query: params } = owner;
    const resource = new DetailResource<readonly string[]>(issueRoot, {
      workspaceID: workspace,
      projectIDs: params.project_id ? [params.project_id] : [],
    });
    const entry: NonNullable<typeof held> = { owner, resource };
    setHeld(entry);
    const run = () => {
      void resource
        .fetch(
          async (read) => {
            entry.proof = read.token;
            const answer = await issueRoot.searchApi.search(workspaceSlug, params);
            const projects = [...new Set(answer.data.map((issue) => issue.project_id))];
            const current = () => {
              // The empty-project watermark also covers a workspace search while its projects were unknown.
              if (!read.isCurrent() || !read.isCurrent(projects)) throw new SessionChangedError();
            };
            current();
            if (
              answer.data.some(
                (issue) =>
                  issue.workspace_id !== workspace ||
                  (params.project_id !== undefined && issue.project_id !== params.project_id)
              )
            )
              throw new Error("Issue search returned a foreign scope");
            const directory = issueRoot.rootStore.projectRoot.project;
            await Promise.all(
              projects.map(async (id) => {
                current();
                if (!directory.getProjectById(id)) await directory.fetchProject(id);
                current();
                if (directory.getProjectById(id)?.workspace_id !== workspace) throw new SessionChangedError();
                issueRoot.assertProjectReadable(workspaceSlug, id);
                const states = issueRoot.rootStore.state;
                if (states.getProjectStates(id) === undefined) await states.fetchProjectStates(id);
                current();
              })
            );
            current();
            if (answer.data.some((issue) => !issueRoot.canSeeIssue(issue))) throw new SessionChangedError();
            issueRoot.acceptListIssues(answer.data, read.token);
            return [...new Set(answer.data.map((issue) => issue.id))].filter(
              (id) => !!issueRoot.issues.getIssueById(id)
            );
          },
          (ids) =>
            ids.flatMap((id) => {
              const issue = issueRoot.issues.getIssueById(id);
              return issue ? [issue.project_id] : [];
            })
        )
        .catch(() => {
          /* The resource owns the current read error. */
        });
    };
    // A new owner hides and disposes the old candidates immediately; only the HTTP start is delayed.
    const timer = params.q ? setTimeout(run, 500) : undefined;
    if (!params.q) run();
    return () => {
      clearTimeout(timer);
      resource.dispose();
    };
  }, [owner]);

  const resource = held?.owner === owner ? held.resource : undefined;
  const value = resource?.value;
  const error = resource?.error;
  return {
    ids: owner.enabled ? (value ?? []) : [],
    proof: owner.enabled && value !== undefined ? held?.proof : undefined,
    loading: owner.enabled && (resource?.loading || (value === undefined && error === undefined)),
    error: owner.enabled && !(error instanceof SessionChangedError) ? error : undefined,
    retry: () => setAttempt((previous) => previous + 1),
  };
}
