/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import type { TNameDescriptionLoader } from "@nerve/types";
import { CustomMenu, EHeaderVariant, Header } from "@nerve/ui";
// components
import { NameDescriptionUpdateStatus } from "@/components/issues/issue-update-status";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// local imports
import type { IntakeSelectionPort } from "../selection-port";
import { INTAKE_ACTION_LABELS, type IntakeActions } from "./intake-actions";
import { IntakeStatusChip } from "./status-chip";

type Props = {
  issueId: string;
  actions: IntakeActions;
  isSubmitting: TNameDescriptionLoader;
  /** The selection, as what shows it gives it: the narrow screen's navigation (R18). */
  selection: IntakeSelectionPort;
};

/**
 * The header of the selected work item of a project's intake, on a narrow screen (P5a spec 2.7, 2.8): the navigation
 * its selection gives (the intake page's: the button that shows or hides the list beside it, the previous and the next
 * work items of the list), the work item's status, whether its name and description are saved, and one menu of every
 * action the action model offers, as the wide header's (IntakeHeader): it holds the status's primary action, its
 * button where the focus lands for it (data-intake-primary, R35), and waits while a triage is out (busy). Drawn only
 * while the work item is held.
 */
export const IntakeMobileHeader = observer(function IntakeMobileHeader(props: Props) {
  const { issueId, actions, isSubmitting, selection } = props;
  const { t } = useTranslation();
  const {
    issue: { getIssueDetailById },
  } = useIssueDetail();
  const detail = getIssueDetailById(issueId);
  if (!detail?.intake) return null;
  return (
    <div className="lg:hidden">
      <Header variant={EHeaderVariant.SECONDARY} className="justify-start gap-2">
        {selection.navigation("narrow")}
        <IntakeStatusChip intake={detail.intake} />
        <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
        <div className="ml-auto" data-intake-primary={actions.primary === undefined ? undefined : true}>
          <CustomMenu
            ariaLabel={t("common.options")}
            customButton={<MoreHorizontalOutline className="size-4" />}
            customButtonClassName={getIconButtonStyling("secondary", "lg")}
            placement="bottom-end"
            disabled={actions.busy}
          >
            {actions.offered.map((action) => (
              <CustomMenu.MenuItem key={action} onClick={(event) => actions.take(action, event.currentTarget)}>
                {t(INTAKE_ACTION_LABELS[action])}
              </CustomMenu.MenuItem>
            ))}
          </CustomMenu>
        </div>
      </Header>
    </div>
  );
});
