/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Workspace } from "@nerve/api-client";
import { useMemo, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "./use-detail-scope";

/** Only the dropdown's pending/error state. Directory values and permission facts remain in their stores. */
export function useDirectoryRead(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  read?: (workspace: Pick<Workspace, "id" | "slug">) => Promise<unknown>
) {
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(workspace?.id, projectId ? [projectId] : []);
  const scope = useDetailScope(workspaceSlug ?? "", projectId, "directory");
  const owner = useMemo(() => ({ request: 0, scope, identity }), [scope, identity]);
  const [state, setState] = useState<{ owner: typeof owner; loading: boolean; error?: unknown }>();
  const load = async () => {
    const current = owner.scope();
    if (!read || !workspace || !current()) return;
    const request = ++owner.request;
    const active = () => current() && request === owner.request;
    setState({ owner, loading: true });
    try {
      const answer = await read(workspace);
      if (!active()) return;
      if (answer === undefined) throw new Error("Directory read was superseded");
      setState({ owner, loading: false });
    } catch (error) {
      if (active())
        setState({ owner, loading: false, error: error instanceof SessionChangedError ? undefined : error });
    }
  };
  return {
    load,
    loading: state?.owner === owner && state.loading,
    error: state?.owner === owner ? state.error : undefined,
  };
}

export function DirectoryReadError({ error, retry }: { error: unknown; retry: () => Promise<void> }) {
  const message = useRefusalMessage();
  const { t } = useTranslation();
  return (
    <div role="alert">
      {message(error)}{" "}
      <button type="button" onClick={() => void retry()}>
        {t("common.retry")}
      </button>
    </div>
  );
}
