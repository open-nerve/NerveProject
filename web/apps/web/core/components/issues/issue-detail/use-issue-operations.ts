/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useMemo } from "react";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { useDetailScope } from "../issue-detail-widgets/use-detail-scope";
import type { TIssueOperations } from "./root";

/** A reported refusal remains a refusal for editors and dialogs waiting for confirmation. */
export function useDetailCommand(workspaceSlug: string, projectId: string | undefined, issueId: string) {
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const refuse = useRefusalToast();
  return useCallback(
    async <T>(command: () => Promise<T>): Promise<T> => {
      const current = capture();
      const inSession = sessionGuard();
      if (!current() || !inSession()) throw new SessionChangedError();
      let result: T;
      try {
        result = await command();
      } catch (error) {
        if (current() && inSession() && !(error instanceof SessionChangedError)) refuse(error);
        throw error;
      }
      if (!current() || !inSession()) throw new SessionChangedError();
      return result;
    },
    [capture, refuse]
  );
}

export function useIssueOperations(
  workspaceSlug: string,
  projectId: string | undefined,
  issueId: string
): TIssueOperations {
  const detail = useIssueDetail();
  const run = useDetailCommand(workspaceSlug, projectId, issueId);
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const refuse = useRefusalToast();
  const { t } = useTranslation();
  return useMemo(
    () => ({
      fetch: async (ws, project, id) => {
        await detail.fetchIssue(ws, project, id);
      },
      update: (ws, project, id, body) => run(() => detail.updateIssue(ws, project, id, body)),
      prepareUpdate: () => {
        const current = capture();
        const inSession = sessionGuard();
        const root = detail.rootIssueStore;
        const permission = root.permissionReads.capture();
        const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
        const authorized = current() && inSession();
        return async (ws, project, id, body) => {
          if (
            !authorized ||
            !root.active ||
            !inSession() ||
            ws !== workspaceSlug ||
            project !== projectId ||
            id !== issueId ||
            !workspace ||
            !root.permissionReads.current(permission, workspace.id, [project], true)
          )
            throw new SessionChangedError();
          try {
            await detail.updateIssue(ws, project, id, body);
          } catch (error) {
            if (current() && inSession() && !(error instanceof SessionChangedError)) refuse(error);
            throw error;
          }
          if (!current() || !inSession()) throw new SessionChangedError();
        };
      },
      restore: async (ws, project, id) => {
        await run(() => detail.rootIssueStore.unarchiveIssue(id, detail.rootIssueStore.scope(ws, project)));
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("issue.restore.success.title"),
          message: t("issue.restore.success.message"),
        });
      },
    }),
    [detail, run, t, capture, refuse, workspaceSlug, projectId, issueId]
  );
}
