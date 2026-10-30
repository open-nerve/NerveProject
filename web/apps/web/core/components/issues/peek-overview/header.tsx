/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, type ComponentType } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
import {
  ArrowNarrowRightOutline,
  DragDropOutline,
  FullScreenPeekOutline,
  LinkOutline,
  ModalPeekOutline,
  SidePeekOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { TNameDescriptionLoader } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { CustomSelect } from "@nerve/ui";
import { copyUrlToClipboard, generateWorkItemLink } from "@nerve/utils";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { WorkItemDetailQuickActions } from "../issue-layouts/quick-action-dropdowns";
import { NameDescriptionUpdateStatus } from "../issue-update-status";
import { IconButton } from "@nerve/propel/icon-button";

export type TPeekModes = "side-peek" | "modal" | "full-screen";

const PEEK_OPTIONS: { key: TPeekModes; icon: ComponentType<{ className?: string }>; i18n_title: string }[] = [
  {
    key: "side-peek",
    icon: SidePeekOutline,
    i18n_title: "common.side_peek",
  },
  {
    key: "modal",
    icon: ModalPeekOutline,
    i18n_title: "common.modal",
  },
  {
    key: "full-screen",
    icon: FullScreenPeekOutline,
    i18n_title: "common.full_screen",
  },
];

export type PeekOverviewHeaderProps = {
  peekMode: TPeekModes;
  setPeekMode: (value: TPeekModes) => void;
  removeRoutePeekId: () => void;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isArchived: boolean;
  disabled: boolean;
  embedIssue: boolean;
  toggleDeleteIssueModal: (value: boolean) => void;
  toggleArchiveIssueModal: (value: boolean) => void;
  toggleEditIssueModal: (value: boolean) => void;
  handleRestoreIssue: () => Promise<void>;
  isSubmitting: TNameDescriptionLoader;
};

export const IssuePeekOverviewHeader = observer(function IssuePeekOverviewHeader(props: PeekOverviewHeaderProps) {
  const {
    peekMode,
    setPeekMode,
    workspaceSlug,
    projectId,
    issueId,
    isArchived,
    disabled,
    embedIssue = false,
    removeRoutePeekId,
    toggleDeleteIssueModal,
    toggleArchiveIssueModal,
    toggleEditIssueModal,
    handleRestoreIssue,
    isSubmitting,
  } = props;
  // ref
  const parentRef = useRef<HTMLDivElement>(null);
  const { t } = useTranslation();
  // store hooks
  const {
    issue: { getIssueById },
    removeIssue,
    archiveIssue,
  } = useIssueDetail();
  const { isMobile } = usePlatformOS();
  const { getProjectIdentifierById } = useProject();
  // derived values
  const issueDetails = getIssueById(issueId);
  const currentMode = PEEK_OPTIONS.find((m) => m.key === peekMode);
  const projectIdentifier = getProjectIdentifierById(issueDetails?.project_id);
  const {
    issues: { removeIssue: removeArchivedIssue },
  } = useIssues(EIssuesStoreType.ARCHIVED);

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issueDetails?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issueDetails?.sequence_id,
    isArchived,
  });

  const handleCopyText = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    copyUrlToClipboard(workItemLink).then(() => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("common.link_copied_to_clipboard"),
      });
      return;
    });
  };

  const handleDeleteIssue = async () => {
    const deleteIssue = issueDetails?.archived_at ? removeArchivedIssue : removeIssue;
    await deleteIssue(workspaceSlug, projectId, issueId);
    removeRoutePeekId();
  };

  const handleArchiveIssue = async () => {
    await archiveIssue(workspaceSlug, projectId, issueId);
    removeRoutePeekId();
  };

  return (
    <div
      className={`relative flex items-center justify-between p-4 ${
        currentMode?.key === "full-screen" ? "border-b border-subtle" : ""
      }`}
    >
      <div className="flex items-center gap-4">
        <Tooltip label={t("common.close_peek_view")} disabled={isMobile}>
          <button type="button" aria-label={t("common.close_peek_view")} onClick={removeRoutePeekId}>
            <ArrowNarrowRightOutline className="h-4 w-4 text-tertiary hover:text-secondary" />
          </button>
        </Tooltip>

        <Tooltip label={t("issue.open_in_full_screen")} disabled={isMobile}>
          <Link aria-label={t("issue.open_in_full_screen")} to={workItemLink} onClick={() => removeRoutePeekId()}>
            <DragDropOutline className="h-4 w-4 text-tertiary hover:text-secondary" />
          </Link>
        </Tooltip>
        {currentMode && embedIssue === false && (
          <div className="flex flex-shrink-0 items-center gap-2">
            <CustomSelect
              value={currentMode.key}
              onChange={(val: TPeekModes) => setPeekMode(val)}
              customButton={
                <Tooltip label={t("common.toggle_peek_view_layout")} disabled={isMobile}>
                  <span>
                    <span className="sr-only">{t("common.toggle_peek_view_layout")}</span>
                    <currentMode.icon className="h-4 w-4 text-tertiary hover:text-secondary" />
                  </span>
                </Tooltip>
              }
            >
              {PEEK_OPTIONS.map((mode) => (
                <CustomSelect.Option key={mode.key} value={mode.key}>
                  <div
                    className={`flex items-center gap-1.5 ${
                      currentMode.key === mode.key ? "text-secondary" : "text-placeholder hover:text-secondary"
                    }`}
                  >
                    <mode.icon className="-my-1 h-4 w-4 flex-shrink-0" />
                    {t(mode.i18n_title)}
                  </div>
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          </div>
        )}
      </div>
      <div className="flex items-center gap-x-4">
        <NameDescriptionUpdateStatus isSubmitting={isSubmitting} />
        <div className="flex items-center gap-2">
          <Tooltip label={t("common.actions.copy_link")} disabled={isMobile}>
            <IconButton
              variant="secondary"
              size="lg"
              onClick={handleCopyText}
              icon={LinkOutline}
              aria-label={t("common.actions.copy_link")}
            />
          </Tooltip>
          {issueDetails && (
            <WorkItemDetailQuickActions
              parentRef={parentRef}
              issue={issueDetails}
              handleDelete={handleDeleteIssue}
              handleArchive={handleArchiveIssue}
              handleRestore={handleRestoreIssue}
              readOnly={disabled}
              toggleDeleteIssueModal={toggleDeleteIssueModal}
              toggleArchiveIssueModal={toggleArchiveIssueModal}
              toggleEditIssueModal={toggleEditIssueModal}
              isPeekMode
            />
          )}
        </div>
      </div>
    </div>
  );
});
