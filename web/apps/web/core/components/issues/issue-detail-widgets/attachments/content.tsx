/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useContext, useLayoutEffect, useState } from "react";
import { observer } from "mobx-react";
import { useContentView } from "@/hooks/use-content-assets";
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import { StoreContext } from "@/lib/store-context";
import { Attachments } from "@/components/assets/attachments/root";
import type { AttachmentResource } from "@/store/asset/attachment-resource";
import type { ContentView } from "@/store/asset/content-view";
const OWNER = "issue-attachments";
export const IssueAttachmentsContent = observer(function IssueAttachmentsContent({
  issueId,
  disabled,
}: {
  issueId: string;
  disabled: boolean;
}) {
  const root = useContext(StoreContext);
  const issue = useIssueDetail().issue.getIssueDetailById(issueId);
  const inputs = useIssueDetailInput();
  const modals = useIssueDetailModals();
  const view = useContentView(
    issue
      ? { kind: "issue", workspaceID: issue.workspace_id, projectID: issue.project_id, issueID: issue.id }
      : undefined
  );
  const [held, setHeld] = useState<{ view: ContentView; resource: AttachmentResource }>();
  useLayoutEffect(() => {
    if (!view?.current) return;
    const resource = root.asset.attachments(view, (receipt) => inputs?.confirm(receipt, OWNER));
    setHeld({ view, resource });
    return () => resource.dispose();
  }, [root, view, inputs]);
  const takeFocus = useCallback(() => inputs?.takeFocus(OWNER), [inputs]);
  const beginCommand = useCallback(() => inputs?.beginCommand(), [inputs]);
  return view?.current && held?.view === view ? (
    <Attachments
      view={view}
      resource={held.resource}
      readOnly={disabled}
      onDialogOpened={modals.retainContentDialog}
      takeFocus={takeFocus}
      beginCommand={beginCommand}
    />
  ) : null;
});
