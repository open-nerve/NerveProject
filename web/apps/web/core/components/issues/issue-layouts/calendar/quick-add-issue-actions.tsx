import type { IssueCreate } from "@nerve/api-client";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";

import { useTranslation } from "@nerve/i18n";
// nerve imports
import { AddOutline } from "@makeplane/propel/icons";
import { EIssueLayoutTypes } from "@nerve/types";
import { CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
// local imports
import { useAddExistingRequest } from "../command-host";
import { QuickAddIssueRoot } from "../quick-add";
import type { CalendarStoreType } from "./base-calendar-root";

type TCalendarQuickAddIssueActions = {
  prePopulatedData?: Partial<IssueDetail>;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
};

export const CalendarQuickAddIssueActions = observer(function CalendarQuickAddIssueActions(
  props: TCalendarQuickAddIssueActions
) {
  const { prePopulatedData, quickAddCallback } = props;
  const { t } = useTranslation();
  // router
  const { projectId } = useParams();
  // states
  const [isOpen, setIsOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  // the page's own list, whose actions add the work items chosen (P7b spec 2.10)
  const storeType = useIssueStoreType() as CalendarStoreType;
  const requestAddExisting = useAddExistingRequest();
  const day = prePopulatedData?.target_date;

  const handleNewIssue = () => {
    setIsOpen(true);
  };
  /**
   * The day's existing work items (P7b spec 2.10, R6, R11): the layout's command host holds the dialog, which outlives
   * the day while the list is checked, and adds those chosen by the page's actions: the project's page gives each the
   * day alone (M4 I4); a cycle's or a module's page gives each the day and joins it to the page in one change, which
   * nerve makes whole or refuses whole, its search hiding too those already in the page's cycle or module. Those of no
   * day are searched for; those starting after the day are hidden.
   */
  const handleExistingIssue = () => {
    if (!requestAddExisting) throw new Error("Adding existing work items requires an IssueListCommandHost");
    if (!day) throw new Error("A day's existing work items require its day");
    requestAddExisting({
      storeType,
      extra: { target_date: day },
      searchParams: { target_date: ["none"] },
      hide: (issue) => !!issue.target_date || !!(issue.start_date && issue.start_date > day),
    });
  };

  if (!projectId) return null;

  return (
    <>
      <QuickAddIssueRoot
        isQuickAddOpen={isOpen}
        setIsQuickAddOpen={setIsOpen}
        layout={EIssueLayoutTypes.CALENDAR}
        prePopulatedData={prePopulatedData}
        quickAddCallback={quickAddCallback}
        customQuickAddButton={
          <div
            className={cn(
              "overflow-hidden rounded-sm bg-layer-transparent hover:bg-layer-transparent-hover md:opacity-0 md:group-hover:opacity-100",
              {
                block: isMenuOpen,
              }
            )}
          >
            <CustomMenu
              placement="bottom-start"
              menuButtonOnClick={() => setIsMenuOpen(true)}
              onMenuClose={() => setIsMenuOpen(false)}
              className="w-full"
              customButtonClassName="w-full"
              customButton={
                <div className="flex w-full items-center gap-x-[6px] rounded-md px-2 py-1.5 text-tertiary hover:text-tertiary">
                  <AddOutline className="h-3.5 w-3.5 flex-shrink-0" />
                  <span className="flex-shrink-0 text-13 font-medium">{t("issue.add.label")}</span>
                </div>
              }
            >
              <CustomMenu.MenuItem onClick={handleNewIssue}>{t("issue.add.label")}</CustomMenu.MenuItem>
              <CustomMenu.MenuItem onClick={handleExistingIssue}>{t("issue.add.existing")}</CustomMenu.MenuItem>
            </CustomMenu>
          </div>
        }
      />
    </>
  );
});
