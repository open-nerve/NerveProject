/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { useRefusalMessage } from "@/hooks/use-refusal-toast";

type Props = {
  /** nerve's refusal of a read, or why nerve could not be reached. */
  error: unknown;
  /** Reads again what failed. */
  onRetry: () => void;
};

/**
 * Why nerve could not read what a page of cycles shows (P5a spec 2.5: their list, their archive, a cycle's page and its
 * preview), in the page's language, and a retry: a read that failed never leaves the page waiting.
 */
export function CyclesReadFailure({ error, onRetry }: Props) {
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  return (
    <div role="alert" className="flex size-full flex-col items-center justify-center gap-3 text-13 text-secondary">
      <span>{refusalMessage(error)}</span>
      <Button variant="secondary" size="sm" onClick={onRetry}>
        {t("common.retry")}
      </Button>
    </div>
  );
}
