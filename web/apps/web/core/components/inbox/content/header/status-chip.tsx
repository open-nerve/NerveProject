/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import {
  ClockOutline,
  CloseCircleOutline,
  CopyOutline,
  TickCircleOutline,
  WarningTriangleOutline,
} from "@makeplane/propel/icons";
// nerve imports
import type { IntakeStatus, IssueIntake } from "@nerve/api-client";
import { INTAKE_STATUS_LABELS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { cn, findHowManyDaysLeft } from "@nerve/utils";
// store
import { effectiveStatus } from "@/store/issue/intake/status";

/** How each status looks: its icon and its colours. */
const LOOKS: Readonly<Record<IntakeStatus, { icon: typeof ClockOutline; className: string }>> = {
  pending: { icon: WarningTriangleOutline, className: "text-warning-primary bg-warning-subtle" },
  snoozed: { icon: ClockOutline, className: "text-placeholder bg-layer-3" },
  accepted: { icon: TickCircleOutline, className: "text-success-primary bg-success-subtle" },
  declined: { icon: CloseCircleOutline, className: "text-danger-primary bg-danger-subtle" },
  duplicate: { icon: CopyOutline, className: "text-secondary bg-layer-3" },
};

/**
 * The status of a work item of a project's intake, by its name (P5a spec 2.7): the status nerve would answer now
 * (effectiveStatus: a snooze whose day has come is pending), a snooze saying the days it has to go.
 */
export function IntakeStatusChip({ intake }: { intake: IssueIntake }) {
  const { t } = useTranslation();
  const status = effectiveStatus(intake, new Date());
  const days = status === "snoozed" ? findHowManyDaysLeft(intake.snoozed_till) : undefined;
  const { icon: Icon, className } = LOOKS[status];
  return (
    <div className={cn("flex items-center gap-1 rounded-sm p-1.5 py-0.5", className)}>
      <Icon width={12} height={12} className="flex-shrink-0" />
      <span className="text-11 font-medium whitespace-nowrap">
        {days === undefined ? t(INTAKE_STATUS_LABELS[status]) : t("inbox_issue.status.snoozed.description", { days })}
      </span>
    </div>
  );
}
