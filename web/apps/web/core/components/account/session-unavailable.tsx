/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// layouts
import DefaultLayout from "@/layouts/default-layout";

type Props = {
  /** Tries again at once. */
  onRetry: () => void;
  /** Whether a retry is scheduled: only then does the page say that it tries again by itself. */
  autoRetry: boolean;
};

/**
 * The page while the session is kept but nerve cannot be reached (M2 design 7.1, 7.4): a refresh failed
 * with 429, 5xx or no network, or the account could not be fetched. It stays on the page the user asked
 * for, never on the sign-in page.
 */
export function SessionUnavailable({ onRetry, autoRetry }: Props) {
  const { t } = useTranslation();
  return (
    <DefaultLayout>
      <div
        role="alert"
        className="relative container mx-auto flex h-full w-full max-w-xl flex-col items-center justify-center gap-4 text-center"
      >
        <h1 className="text-h4-semibold text-primary">{t("auth.session_unavailable.title")}</h1>
        <p className="text-body-sm-regular text-secondary">
          {t(autoRetry ? "auth.session_unavailable.description_auto_retry" : "auth.session_unavailable.description")}
        </p>
        <Button variant="primary" onClick={onRetry}>
          {t("auth.session_unavailable.retry")}
        </Button>
      </div>
    </DefaultLayout>
  );
}
