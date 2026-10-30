/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { followInSession } from "@/lib/in-session";
import { useDetailScope } from "./use-detail-scope";

export function DetailResourceError({
  error,
  retry,
  workspaceSlug,
  projectId,
  issueId,
}: {
  error: unknown;
  retry: () => Promise<unknown>;
  workspaceSlug: string;
  projectId: string | undefined;
  issueId: string;
}) {
  const { t } = useTranslation();
  const message = useRefusalMessage();
  const scope = useDetailScope(workspaceSlug, projectId, issueId);
  if (!error || error instanceof SessionChangedError) return null;
  const reread = async () => {
    const current = scope();
    if (!current()) return;
    // The resource owns the current failure and keeps it visible if this read is refused again.
    await followInSession(retry, { failed: () => undefined });
  };
  return (
    <div role="alert">
      {message(error)}{" "}
      <button type="button" onClick={() => void reread()}>
        {t("common.retry")}
      </button>
    </div>
  );
}
