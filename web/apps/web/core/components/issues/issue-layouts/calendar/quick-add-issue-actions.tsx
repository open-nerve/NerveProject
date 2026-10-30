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
// components
import {
  ExistingIssuesListModal,
  type SelectionSubmitResult,
} from "@/components/core/modals/existing-issues-list-modal";
// hooks
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { QuickAddIssueRoot } from "../quick-add";

type TCalendarQuickAddIssueActions = {
  prePopulatedData?: Partial<IssueDetail>;
  quickAddCallback?: (projectId: string | null | undefined, data: IssueCreate) => Promise<Issue | undefined>;
  onOpen?: () => void;
};

export const CalendarQuickAddIssueActions = observer(function CalendarQuickAddIssueActions(
  props: TCalendarQuickAddIssueActions
) {
  const { prePopulatedData, quickAddCallback, onOpen } = props;
  const { t } = useTranslation();
  // router
  const { workspaceSlug, projectId } = useParams();
  // states
  const [isOpen, setIsOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isExistingIssueModalOpen, setIsExistingIssueModalOpen] = useState(false);
  const { updateIssue } = useIssueDetail();
  const handleAddIssuesToView = async (ids: readonly string[]): Promise<SelectionSubmitResult> => {
    if (!workspaceSlug || !projectId || !prePopulatedData?.target_date) throw new SessionChangedError();
    const unique = [...new Set(ids)];
    const answers = await Promise.allSettled(
      unique.map((id) => updateIssue(workspaceSlug, projectId, id, { target_date: prePopulatedData.target_date }))
    );
    return {
      confirmedIds: unique.filter((_, index) => answers[index]!.status === "fulfilled"),
      failures: answers.flatMap((answer, index) =>
        answer.status === "rejected" ? [{ id: unique[index]!, error: answer.reason as unknown }] : []
      ),
    };
  };

  const handleNewIssue = () => {
    setIsOpen(true);
    if (onOpen) onOpen();
  };
  const handleExistingIssue = () => {
    setIsExistingIssueModalOpen(true);
  };

  if (!projectId) return null;

  return (
    <>
      {workspaceSlug && projectId && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isOpen={isExistingIssueModalOpen}
          handleClose={() => setIsExistingIssueModalOpen(false)}
          key={prePopulatedData?.target_date}
          searchParams={{ target_date: ["none"] }}
          handleOnSubmit={handleAddIssuesToView}
          shouldHideIssue={(issue) =>
            !!issue.target_date ||
            !!(issue.start_date && prePopulatedData?.target_date && issue.start_date > prePopulatedData.target_date)
          }
        />
      )}
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
