/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef, useState } from "react";
import { ApiError } from "@/lib/api-error";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import type { RetainedForm } from "../issues/detail-input-state";
/** Pure input and outcome bookkeeping; callers retain their own canonical view and command guards. */
export function useCommentInput(key: string, baseHTML: string) {
  const retained = useIssueDetailInput()?.form(key);
  const [value, setValue] = useState<RetainedForm>(
    () => retained?.read() ?? { html: baseHTML, baseHTML, attachmentIDs: [], revision: 0 }
  );
  const latest = useRef({ retained, value });
  latest.current = { retained, value };
  const update = useCallback((change: (previous: RetainedForm) => RetainedForm) => {
    const next = change(latest.current.value);
    latest.current.value = next;
    latest.current.retained?.store(next);
    setValue(next);
  }, []);
  const change = useCallback(
    (html: string) =>
      update((previous) => ({
        ...previous,
        html,
        revision: previous.revision + 1,
        error: undefined,
        rejectedRevision: undefined,
      })),
    [update]
  );
  const select = useCallback(
    (attachmentIDs: string[]) => update((previous) => ({ ...previous, attachmentIDs })),
    [update]
  );
  const begin = () => update((previous) => ({ ...previous, unknown: true, verified: false, error: undefined }));
  const failed = (error: unknown) =>
    update((previous) => ({
      ...previous,
      error,
      rejectedRevision: previous.revision,
      unknown: !(error instanceof ApiError && error.problem),
      verified: false,
    }));
  const confirmed = (html = baseHTML, focusAfterRead = false) =>
    update((previous) => ({
      html,
      baseHTML: html,
      attachmentIDs: [],
      revision: previous.revision + 1,
      editing: false,
      focusAfterRead,
    }));
  const focused = useCallback(() => update((previous) => ({ ...previous, focusAfterRead: false })), [update]);
  const conflict = value.html !== value.baseHTML && baseHTML !== value.baseHTML && value.html !== baseHTML;
  return {
    value,
    change,
    select,
    begin,
    failed,
    confirmed,
    conflict,
    focused,
    verify: () => update((previous) => ({ ...previous, verified: true })),
    allowResend: () =>
      update((previous) =>
        previous.verified ? { ...previous, unknown: false, verified: false, error: undefined } : previous
      ),
    keep: () => update((previous) => ({ ...previous, baseHTML })),
  };
}
