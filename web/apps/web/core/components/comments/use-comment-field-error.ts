/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLayoutEffect } from "react";
import { useTranslation } from "@nerve/i18n";
import { fieldErrorKeys } from "@/lib/error-messages";
/** Restore field semantics without replaying a toast or a command when a preserved form remounts. */
export function useCommentFieldError(
  error: unknown,
  setError: (field: "comment_html", error: { type: string; message: string }) => void
) {
  const { t } = useTranslation();
  const key = fieldErrorKeys(error).comment_html;
  const message = key ? t(key) : undefined;
  useLayoutEffect(() => {
    if (message) setError("comment_html", { type: "server", message });
  }, [message, setError]);
}
