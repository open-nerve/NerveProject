/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { useRefusalMessage } from "@/hooks/use-refusal-toast";

type Props = {
  /** nerve's refusal of the read again, or why nerve could not be reached. */
  error: unknown;
  /** Reads the page's preferences again. */
  onRetry: () => void;
};

/**
 * The one line above a page's list of work items (the project's, a cycle's, a module's: P6b spec 2.8, R4) that says
 * nerve could not read the page's own preferences again once it had given them, why, in the page's language, and a
 * retry: the list stays as it shows under the preferences read, which a change of them elsewhere does not reach before
 * a read of them succeeds.
 */
export function PreferencesRereadFailure({ error, onRetry }: Props) {
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  return (
    <div
      role="alert"
      className="flex w-full flex-shrink-0 flex-wrap items-center gap-x-2 gap-y-1 border-b border-subtle bg-layer-1 px-4 py-2 text-13 text-secondary"
    >
      <span>{t("issue.layouts.preferences_reread_failed")}</span>
      <span>{refusalMessage(error)}</span>
      <Button variant="secondary" size="sm" onClick={onRetry}>
        {t("common.retry")}
      </Button>
    </div>
  );
}
