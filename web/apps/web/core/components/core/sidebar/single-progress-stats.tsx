/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { cn, progressPercent } from "@nerve/utils";

type TSingleProgressStatsProps = {
  title: ReactNode;
  completed: number;
  total: number;
  /** What choosing the row does (P5b spec 2.8); without it the row is plain. */
  onClick?: () => void;
  /** Whether the row's value is chosen, for a row whose choice toggles it; said as the button's pressed state. */
  selected?: boolean;
};

/**
 * A row of a progress's breakdown: its title, and the share completed of its total, in percent as a progress counts
 * it (progressPercent, the one count of M6 design 3.5: a row's total is what it counts, none of it cancelled). A row
 * that can be chosen is a button, named by the row (its title and its share), that says whether its value is chosen
 * when its choice toggles it.
 */
export function SingleProgressStats({ title, completed, total, onClick, selected }: TSingleProgressStatsProps) {
  const { t } = useTranslation();
  const percent = progressPercent({ total_issues: total, completed_issues: completed, cancelled_issues: 0 });
  const content = (
    <>
      <div className="w-4/6">{title}</div>
      <div className="flex w-2/6 items-center justify-end gap-1 px-2">
        <div className="flex h-5 items-center justify-center gap-1">
          <span className="w-8 text-right">{percent}%</span>
        </div>
        <span>{t("distribution.of_total", { total })}</span>
      </div>
    </>
  );
  const row = "flex w-full items-center justify-between gap-4 rounded-xs p-1 text-11";
  if (!onClick) return <div className={row}>{content}</div>;
  return (
    <button
      type="button"
      aria-pressed={selected}
      className={cn(row, "cursor-pointer text-left hover:bg-surface-2", { "bg-layer-1": selected })}
      onClick={onClick}
    >
      {content}
    </button>
  );
}
