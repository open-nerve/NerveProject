/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import {
  CloseCircleFilled,
  LinkOutline,
  MoreHorizontalOutline,
  NewTabOutline,
  TickCircleFilled,
} from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button, getButtonStyling } from "@nerve/propel/button";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import type { TNameDescriptionLoader } from "@nerve/types";
import { ControlLink, CustomMenu, Row } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { NameDescriptionUpdateStatus } from "@/components/issues/issue-update-status";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
// local imports
import { IntakeNavigation } from "./navigation";
import { IntakeStatusChip } from "./status-chip";
import { INTAKE_ACTION_LABELS, type IntakeAction, type IntakeActions } from "./use-intake-actions";

/** The actions the wide header offers in its menu, in its order: the others are buttons. */
const MENU_ACTIONS: readonly IntakeAction[] = ["snooze", "unsnooze", "resnooze", "duplicate", "delete"];

type Props = {
  projectId: string;
  issueId: string;
  actions: IntakeActions;
  isSubmitting: TNameDescriptionLoader;
};

/**
 * The header of the selected work item of a project's intake, on a wide screen (P5a spec 2.7, 2.8): its identifier in
 * the route's project, its status, whether its name and description are saved, the previous and the next work items of
 * the list (IntakeNavigation), and the actions the action model offers: accepting and declining, copying its link and
 * opening it as buttons, the others in a menu; accepting is the status's primary action (data-intake-primary, R35).
 * Each control gives the action model itself, where the focus goes back once the action ends; the triages and the menu
 * wait while one is out (busy). Drawn only while the work item is held, so that a triage's read again withdraws it
 * (R34).
 */
export const IntakeHeader = observer(function IntakeHeader({ projectId, issueId, actions, isSubmitting }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const { getProjectById } = useProject();
  const detail = getIssueDetailById(issueId);
  if (!detail?.intake) return null;
  const { offered, take, link, busy } = actions;
  const inMenu = MENU_ACTIONS.filter((action) => offered.includes(action));
  return (
    <Row className="relative z-15 hidden h-full w-full items-center justify-between gap-2 border-b border-subtle bg-surface-1 lg:flex">
      <div className="flex items-center gap-4">
        <h3 className="flex-shrink-0 text-14 font-medium text-tertiary">
          {getProjectById(projectId)?.identifier}-{detail.sequence_id}
        </h3>
        <IntakeStatusChip intake={detail.intake} />
        <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <IntakeNavigation projectId={projectId} issueId={issueId} />
        {offered.includes("accept") && (
          <Button
            variant="secondary"
            size="lg"
            disabled={busy}
            data-intake-primary
            onClick={(event) => take("accept", event.currentTarget)}
          >
            <TickCircleFilled className="size-4 shrink-0 text-success-secondary" />
            {t(INTAKE_ACTION_LABELS.accept)}
          </Button>
        )}
        {offered.includes("decline") && (
          <Button
            variant="secondary"
            size="lg"
            disabled={busy}
            onClick={(event) => take("decline", event.currentTarget)}
          >
            <CloseCircleFilled className="size-4 shrink-0 text-danger-secondary" />
            {t(INTAKE_ACTION_LABELS.decline)}
          </Button>
        )}
        <Button
          variant="secondary"
          size="lg"
          prependIcon={<LinkOutline className="h-2.5 w-2.5" />}
          onClick={(event) => take("copy", event.currentTarget)}
        >
          {t(INTAKE_ACTION_LABELS.copy)}
        </Button>
        {offered.includes("open") && (
          <ControlLink
            href={link}
            onClick={() => void navigate(link)}
            target="_self"
            className={cn(getButtonStyling("secondary", "lg"), "gap-1.5")}
          >
            <NewTabOutline className="h-2.5 w-2.5" />
            {t(INTAKE_ACTION_LABELS.open)}
          </ControlLink>
        )}
        {inMenu.length > 0 && (
          <CustomMenu
            ariaLabel={t("common.options")}
            customButton={<MoreHorizontalOutline className="size-4" />}
            customButtonClassName={getIconButtonStyling("secondary", "lg")}
            placement="bottom-end"
            disabled={busy}
          >
            {inMenu.map((action) => (
              <CustomMenu.MenuItem key={action} onClick={(event) => take(action, event.currentTarget)}>
                {t(INTAKE_ACTION_LABELS[action])}
              </CustomMenu.MenuItem>
            ))}
          </CustomMenu>
        )}
      </div>
    </Row>
  );
});
