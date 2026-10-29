/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";

export function EditorMentionsError({ error, retry }: { error: unknown; retry: () => void }) {
  const { t } = useTranslation();
  const message = useRefusalMessage();
  if (error instanceof SessionChangedError) return null;
  return (
    <div role="alert">
      {message(error)}{" "}
      <button type="button" onClick={retry}>
        {t("common.retry")}
      </button>
    </div>
  );
}
