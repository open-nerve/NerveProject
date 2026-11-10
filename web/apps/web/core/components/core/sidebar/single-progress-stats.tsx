/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { progressPercent } from "@nerve/utils";

type TSingleProgressStatsProps = {
  title: ReactNode;
  completed: number;
  total: number;
};

/**
 * A row of a progress's breakdown: its title, and the share completed of its total, in percent as a progress counts
 * it (progressPercent, the one count of M6 design 3.5: a row's total is what it counts, none of it cancelled).
 */
export function SingleProgressStats({ title, completed, total }: TSingleProgressStatsProps) {
  const { t } = useTranslation();
  const percent = progressPercent({ total_issues: total, completed_issues: completed, cancelled_issues: 0 });
  return (
    <div className="flex w-full items-center justify-between gap-4 rounded-xs p-1 text-11">
      <div className="w-4/6">{title}</div>
      <div className="flex w-2/6 items-center justify-end gap-1 px-2">
        <div className="flex h-5 items-center justify-center gap-1">
          <span className="w-8 text-right">{percent}%</span>
        </div>
        <span>{t("distribution.of_total", { total })}</span>
      </div>
    </div>
  );
}
