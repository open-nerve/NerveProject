/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { Issue } from "@nerve/api-client";
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TDeDupeIssue } from "@nerve/types";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "./issue-detail-widgets/use-detail-scope";

type Props = {
  data?: Issue | TDeDupeIssue;
  dataId?: string | null | undefined;
  handleClose: () => void;
  isOpen: boolean;
  onSubmit: () => Promise<void>;
  onArchived?: () => void;
};
type Snapshot = { id: string; projectId: string; identifier: string | undefined; sequence: number };

export const ArchiveIssueModal = observer(function ArchiveIssueModal(props: Props) {
  const { data, dataId, ...confirmation } = props;
  const { workspaceSlug } = useParams();
  const { getProjectById } = useProject();
  const { issueMap } = useIssues();
  const issue = data ?? (dataId ? issueMap[dataId] : undefined);
  if (!issue || !workspaceSlug || !issue.project_id) return null;
  return (
    <ArchiveConfirmation
      key={JSON.stringify([workspaceSlug, issue.project_id, issue.id, props.isOpen])}
      {...confirmation}
      workspaceSlug={workspaceSlug}
      issue={{
        id: issue.id,
        projectId: issue.project_id,
        identifier: getProjectById(issue.project_id)?.identifier,
        sequence: issue.sequence_id,
      }}
    />
  );
});

function ArchiveConfirmation({
  issue,
  workspaceSlug,
  isOpen,
  handleClose,
  onSubmit,
  onArchived,
}: Omit<Props, "data" | "dataId"> & { workspaceSlug: string; issue: Snapshot }) {
  // One open owns its displayed identity and pending request, even if the canonical issue changes meanwhile.
  const [snapshot] = useState(issue);
  const [isArchiving, setIsArchiving] = useState(false);
  const sending = useRef(false);
  const capture = useDetailScope(workspaceSlug, snapshot.projectId, snapshot.id);
  const refusal = useRefusalToast();
  const { t } = useTranslation();
  const onClose = () => {
    if (!sending.current) handleClose();
  };
  const handleArchiveIssue = async () => {
    const current = capture();
    if (!isOpen || !current() || sending.current) return;
    sending.current = true;
    setIsArchiving(true);
    await followInSession(async () => onSubmit(), {
      done: () => {
        if (!current()) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("issue.archive.success.label"),
          message: t("issue.archive.success.message"),
        });
        handleClose();
        onArchived?.();
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError)) refusal(error);
      },
    });
    if (current()) {
      sending.current = false;
      setIsArchiving(false);
    }
  };
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isArchiving ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.LG}
    >
      <div className="px-5 py-4">
        <ModalTitle className="text-18 font-medium 2xl:text-20">
          {t("issue.archive.label")} {snapshot.identifier} {snapshot.sequence}
        </ModalTitle>
        <p className="mt-3 text-13 text-secondary">{t("issue.archive.confirm_message")}</p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" disabled={isArchiving} onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="primary" size="lg" onClick={() => void handleArchiveIssue()} loading={isArchiving}>
            {isArchiving ? t("common.archiving") : t("common.archive")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
