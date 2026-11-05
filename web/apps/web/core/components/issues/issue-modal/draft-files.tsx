/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import type { EditorImages } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import { StoreContext } from "@/lib/store-context";
import { useContentView } from "@/hooks/use-content-assets";
import { useEditorAssets } from "@/hooks/editor/use-editor-assets";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { Attachments } from "@/components/assets/attachments/root";
import { DraftPreparations } from "@/components/assets/attachments/preparations";
import type { AttachmentResource } from "@/store/asset/attachment-resource";
import type { ContentView } from "@/store/asset/content-view";
import type { DraftPreparationsResource } from "@/store/asset/draft-preparations-resource";

/** A move or publish was not sent: a transfer of the draft is still on the wire. */
export class TransferPendingError extends Error {
  constructor() {
    super("A draft transfer is still in progress");
    this.name = "TransferPendingError";
  }
}
/** The content lifetime of a stored draft in the form. Moving or publishing the draft retires it first. */
export type DraftContent = {
  view: ContentView | undefined;
  images: EditorImages | undefined;
  /** No attachment transfer of the current lifetime is on the wire; editor images report their own readiness. */
  settled(): boolean;
  /** Ends the lifetime before the request leaves, so no queued or batched work of it can continue. */
  retire(): void;
  /** After a refused command, a fresh lifetime reads and authorizes again. */
  restore(): void;
  onBusyChange(busy: boolean): void;
};
export function useDraftContent(workspaceSlug: string, draftID: string | undefined): DraftContent {
  const drafts = useWorkspaceDraftIssues();
  const draft = draftID ? drafts.getIssueById(draftID, workspaceSlug) : undefined;
  const workspace = useWorkspace().getWorkspaceBySlug(workspaceSlug);
  const [retired, setRetired] = useState(false);
  const busy = useRef<{ view?: ContentView; busy: boolean }>({ busy: false });
  const view = useContentView(
    !retired && draft && workspace && draft.workspace_id === workspace.id
      ? { kind: "draft", workspaceID: workspace.id, projectID: draft.project_id, draftID: draft.id }
      : undefined
  );
  const images = useEditorAssets(view);
  return {
    view,
    images,
    settled: () => !(view && busy.current.view === view && busy.current.busy),
    retire: () => {
      setRetired(true);
      view?.dispose();
    },
    restore: () => setRetired(false),
    onBusyChange: (value) => {
      busy.current = { view, busy: value };
    },
  };
}
/** Attachments and the author's unfinished files of one draft lifetime; both end with it. */
export const DraftFiles = observer(function DraftFiles({
  content,
  publishBlocked,
}: {
  content: DraftContent;
  publishBlocked: boolean;
}) {
  const root = useContext(StoreContext);
  const { t } = useTranslation();
  const { view } = content;
  const [held, setHeld] = useState<{
    view: ContentView;
    attachments: AttachmentResource;
    preparations: DraftPreparationsResource;
  }>();
  useLayoutEffect(() => {
    if (!view?.current) return;
    const attachments = root.asset.attachments(view);
    const preparations = root.asset.draftPreparations(view);
    setHeld({ view, attachments, preparations });
    return () => {
      attachments.dispose();
      preparations.dispose();
    };
  }, [root, view]);
  if (!view?.current || held?.view !== view) return null;
  return (
    <section aria-label={t("content_files.title")} className="space-y-3">
      <Attachments view={view} resource={held.attachments} onBusyChange={content.onBusyChange} />
      {publishBlocked && (
        <p role="alert" className="text-13 text-danger-primary">
          {t("content_files.draft.publish_blocked")}
        </p>
      )}
      <DraftPreparations resource={held.preparations} />
    </section>
  );
});
