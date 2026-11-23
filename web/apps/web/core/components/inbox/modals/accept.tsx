/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useMemo } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { IssueDetail } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// components
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";

type Props = {
  projectId: string;
  /** The work item as held when the modal opened: the form starts at it, whatever reads it again meanwhile. */
  detail: IssueDetail;
  /** Whether nerve has accepted it, by this modal or another's acceptance (R29). */
  accepted: boolean;
  /** Accepts the work item (the action model's), before the form's update: a refusal rejects, and stops the update. */
  onAccept: () => Promise<void>;
  /** The modal's close: once its update is confirmed, or a cancel. */
  onClose: () => void;
};

/**
 * The accept modal of a work item of a project's intake (P5a spec 2.8, R12, R29; re-94 I2): M4's issue modal, through
 * its props alone, its form at the work item as held when it opened but its state, which starts at the project's
 * default state (the work item's triage state is in no catalog; nerve moves an accepted work item to the default
 * state, so the form sends a state only when the user picks one). Its submit accepts the work item first
 * (beforeFormSubmit: onAccept), then sends what the form changed as M4's update; once the work item is accepted, the
 * modal says so, its changes not saved until that update is, and its button saves them.
 */
export const IntakeAcceptModal = observer(function IntakeAcceptModal(props: Props) {
  const { projectId, detail, accepted, onAccept, onClose } = props;
  const { t } = useTranslation();
  const { getProjectById } = useProject();
  const { getProjectStates } = useProjectState();
  const start = getProjectStates(projectId)?.find((state) => state.default)?.id;
  const data = useMemo(() => ({ ...detail, state_id: start ?? detail.state_id }), [detail, start]);
  const value = `${getProjectById(projectId)?.identifier}-${detail.sequence_id}`;
  return (
    <CreateUpdateIssueModal
      data={data}
      isOpen
      onClose={onClose}
      beforeFormSubmit={onAccept}
      withDraftIssueWrapper={false}
      fetchIssueDetails={false}
      showActionItemsOnUpdate
      modalTitle={
        accepted ? t("inbox_issue.modals.accept.accepted", { value }) : t("inbox_issue.actions.move", { value })
      }
      primaryButtonText={
        accepted
          ? { default: t("update"), loading: t("updating") }
          : { default: t("add_to_project"), loading: t("adding") }
      }
    />
  );
});
