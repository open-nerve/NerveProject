/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useContext, useLayoutEffect, useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { StoreContext } from "@/lib/store-context";
import { useIssueDetail, useOptionalIssueDetailModals } from "@/hooks/store/use-issue-detail";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import type { ContentView } from "@/store/asset/content-view";
import type { AttachmentResource } from "@/store/asset/attachment-resource";
import { Attachments } from "@/components/assets/attachments/root";
import { useCommentView } from "./use-comment-content";
type Props = {
  workspaceSlug: string;
  projectId: string | undefined;
  issueId: string;
  commentId: string;
  disabled: boolean;
};
export const CommentAttachments = observer(function CommentAttachments(props: Props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const detail = useIssueDetail();
  if (!props.projectId || !detail.comment.getCommentById(props.issueId, props.commentId)) return null;
  return (
    <section className="space-y-2">
      <button type="button" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {t("content_files.title")}
      </button>
      {open && <CommentAttachmentContent {...props} />}
    </section>
  );
});
const CommentAttachmentContent = observer(function CommentAttachmentContent(props: Props) {
  const root = useContext(StoreContext);
  const modals = useOptionalIssueDetailModals();
  const inputs = useIssueDetailInput();
  const view = useCommentView(props.workspaceSlug, props.projectId, props.issueId, props.commentId);
  const [held, setHeld] = useState<{ view: ContentView; resource: AttachmentResource }>();
  const owner = `comment-attachments:${props.commentId}`;
  useLayoutEffect(() => {
    if (!view?.current) return;
    const resource = root.asset.attachments(view, (receipt) => inputs?.confirm(receipt, owner));
    setHeld({ view, resource });
    return () => resource.dispose();
  }, [root, view, inputs, owner]);
  const takeFocus = useCallback(() => inputs?.takeFocus(owner), [inputs, owner]);
  const beginCommand = useCallback(() => inputs?.beginCommand(), [inputs]);
  return view?.current && held?.view === view ? (
    <Attachments
      view={view}
      resource={held.resource}
      readOnly={props.disabled}
      onDialogOpened={modals?.retainContentDialog}
      takeFocus={takeFocus}
      beginCommand={beginCommand}
    />
  ) : null;
});
