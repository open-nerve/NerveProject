/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { ArrowNarrowRightOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";

/**
 * The close of a work item of a project's intake embedded in the notifications page's preview (P5b spec 2.5, R17): a
 * button named Close, where the intake page's headers draw their navigation, which closes the embed (its selection's
 * close).
 */
export function IntakeEmbedClose({ onClose, className }: { onClose: () => void; className?: string }) {
  const { t } = useTranslation();
  return (
    <button type="button" aria-label={t("close")} onClick={onClose} className={cn("flex-shrink-0", className)}>
      <ArrowNarrowRightOutline className="h-4 w-4 text-tertiary hover:text-secondary" />
    </button>
  );
}
