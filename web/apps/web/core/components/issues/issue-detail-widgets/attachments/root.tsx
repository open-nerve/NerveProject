/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { Collapsible } from "@makeplane/propel/components/collapsible";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { IssueAttachmentsContent } from "./content";
export const IssueAttachments = observer(function IssueAttachments({
  issueId,
  disabled,
}: {
  issueId: string;
  disabled: boolean;
}) {
  const detail = useIssueDetail();
  const { t } = useTranslation();
  const issue = detail.issue.getIssueDetailById(issueId);
  const open = detail.openWidgets.includes("attachments");
  if (!issue) return null;
  return (
    <Collapsible
      open={open}
      onOpenChange={() => detail.toggleOpenWidget("attachments")}
      trigger={
        <span className="inline-flex items-center gap-2">
          {t("content_files.title")}
          <span className="text-tertiary">{issue.attachment_count}</span>
        </span>
      }
    >
      {open && <IssueAttachmentsContent issueId={issueId} disabled={disabled} />}
    </Collapsible>
  );
});
