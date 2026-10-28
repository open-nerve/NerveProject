/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useEffect, useRef } from "react";
import { useTranslation } from "@nerve/i18n";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { pagePartKey, type PagePart } from "@/lib/paged-groups";
import type { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";

/** One attempt per visible stale period; only explicit retry or a new mount retries a refusal. */
export function useIssueListRefresh(list: IssueListBinding | undefined, visible: PagePart[]) {
  const toastRefusal = useRefusalToast();
  const owner = useRef(list);
  owner.current = list;
  const reported = useRef<unknown>(undefined);
  const attempts = useRef<{ list: IssueListBinding | undefined; parts: Set<string> }>({ list, parts: new Set() });
  const stale = visible.filter((part) => list?.group(part.group, part.subgroup)?.stale);
  const error = list?.error;
  const current = useCallback(() => !!list?.root.active && owner.current === list, [list]);
  const show = useCallback(
    (reason: unknown) => {
      if (current() && reported.current !== reason) {
        reported.current = reason;
        toastRefusal(reason);
      }
    },
    [current, toastRefusal]
  );
  const run = useCallback(
    async (parts: PagePart[], explicit = false) => {
      if (!current() || !list || (!parts.length && !explicit)) return;
      await followInSession<unknown>(
        () =>
          explicit && list.error ? list.retryFailed() : parts.length ? list.refreshStale(parts) : list.fetchFirst(),
        { failed: show }
      );
    },
    [current, list, show]
  );
  const retry = () => run(stale, true);
  useEffect(
    () => () => {
      owner.current = undefined;
    },
    [list]
  );
  useEffect(() => {
    owner.current = list;
    if (!list || !current()) return;
    if (attempts.current.list !== list) {
      attempts.current = { list, parts: new Set() };
      reported.current = undefined;
    }
    if (error && reported.current !== error) show(error);
    for (const key of attempts.current.parts) {
      const [group, subgroup] = JSON.parse(key) as [string, string | null];
      if (!list.group(group, subgroup ?? undefined)?.stale) attempts.current.parts.delete(key);
    }
    const parts = stale.filter((part) => !attempts.current.parts.has(pagePartKey(part)));
    // A refusal defers other parts; their attempt starts only when a read starts.
    if (parts.length && !error) {
      for (const part of parts) attempts.current.parts.add(pagePartKey(part));
      void run(parts);
    }
  }, [list, error, current, run, stale, show]);
  return { retry, failed: !!error };
}

/** The layout owns an explicit retry; refusals otherwise stay put without an automatic loop. */
export function IssueListRetry({ retry, failed }: { retry: () => Promise<void>; failed: boolean }) {
  const { t } = useTranslation();
  return failed ? (
    <button type="button" className="px-3 py-2 text-13 text-danger-primary" onClick={() => void retry()}>
      {t("common.retry")}
    </button>
  ) : null;
}
