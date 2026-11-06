/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useEffect, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { StoreContext } from "@/lib/store-context";
import { errorMessageKey } from "@/lib/error-messages";
import { useContentUpload } from "@/hooks/use-content-assets";
import type { ContentView } from "@/store/asset/content-view";
import type { AttachmentResource } from "@/store/asset/attachment-resource";
import { AttachmentPicker } from "./picker";
import { AttachmentList } from "./list";
import type { AttachmentRow } from "./item";
import { DeleteAttachmentDialog } from "./delete-dialog";
export const Attachments = observer(function Attachments({
  view,
  resource,
  readOnly = false,
  onDialogOpened,
  onBusyChange,
  takeFocus,
  beginCommand,
}: {
  view: ContentView;
  resource: AttachmentResource;
  readOnly?: boolean;
  onDialogOpened?(): () => void;
  /** Whether a transfer is on the wire; a host that moves or publishes must wait for it. */
  onBusyChange?(busy: boolean): void;
  /** The command a host re-read confirmed while it unmounted this widget, once, so that the focus it lost returns. */
  takeFocus?(): "uploaded" | "deleted" | undefined;
  /** A transfer or deletion starts: the host's notice of an earlier result ends. */
  beginCommand?(): void;
}) {
  const { t } = useTranslation();
  const root = useContext(StoreContext);
  const upload = useContentUpload(readOnly ? undefined : view, "attachment");
  const refresh = useRef<HTMLButtonElement>(null);
  const picker = useRef<HTMLInputElement>(null);
  const [selection, setSelection] = useState<{ resource: AttachmentResource; row: AttachmentRow }>();
  useEffect(() => {
    void resource.load().catch(() => undefined);
  }, [resource]);
  const [returning, setReturning] = useState<"uploaded" | "deleted">();
  useLayoutEffect(() => {
    // Only the focus the remount lost returns: a control chosen meanwhile keeps it, and the notice stays.
    if (!view.current || !focusLost()) return;
    const kind = takeFocus?.();
    if (kind) setReturning(kind);
  }, [view, takeFocus]);
  const pickerExpected = !readOnly && view.canWrite("attachment");
  useLayoutEffect(() => {
    if (!returning) return;
    const target = returning === "uploaded" && pickerExpected ? picker.current : refresh.current;
    // The picker mounts with its upload session, a render after this widget; the reload control waits for the read.
    if (!target || target.disabled) return;
    setReturning(undefined);
    if (focusLost()) target.focus({ preventScroll: true });
  }, [returning, upload, pickerExpected, resource.loading]);
  if (!view.current) return null;
  const selected = selection?.resource === resource ? selection.row : undefined;
  return (
    <div className="space-y-3">
      {upload && (
        <AttachmentPicker
          input={picker}
          upload={upload}
          onUploaded={(asset) => resource.afterUpload(asset)}
          onBusyChange={(busy) => {
            if (busy) beginCommand?.();
            onBusyChange?.(busy);
          }}
        />
      )}
      <div className="flex items-center justify-between">
        <p aria-live="polite" className="text-13 text-secondary">
          {resource.loading ? t("content_files.reading") : resource.rows.length === 0 ? t("content_files.empty") : null}
        </p>
        <Button
          ref={refresh}
          type="button"
          variant="secondary"
          disabled={resource.loading}
          onClick={() => {
            void resource.retryRead().catch(() => undefined);
          }}
        >
          {t("content_files.refresh")}
        </Button>
      </div>
      {resource.error !== undefined && (
        <p role="alert" className="text-13 text-danger-primary">
          {t(errorMessageKey(resource.error))}
        </p>
      )}
      {resource.uncertainIDs.length > 0 && (
        <p className="text-13 text-secondary">{t("content_files.unknown_delete")}</p>
      )}
      <AttachmentList
        rows={resource.rows}
        download={(id) => root.asset.content(view).download(id)}
        remove={
          !readOnly && view.canRemoveAttachment()
            ? (row) => {
                refresh.current?.focus();
                setSelection({ resource, row });
              }
            : undefined
        }
      />
      {resource.nextCursor && (
        <Button
          type="button"
          variant="secondary"
          disabled={resource.loading}
          onClick={() => {
            void resource.more().catch(() => undefined);
          }}
        >
          {t("content_files.more")}
        </Button>
      )}
      {selected && (
        <DeleteAttachmentDialog
          key={selected.id}
          onDialogOpened={onDialogOpened}
          view={view}
          file={selected}
          uncertain={resource.uncertainIDs.includes(selected.id)}
          remove={(id) => {
            beginCommand?.();
            return resource.remove(id);
          }}
          verify={() => resource.verifyDeletion()}
          onClose={() => setSelection(undefined)}
        />
      )}
    </div>
  );
});
const focusLost = () => !document.activeElement || document.activeElement === document.body;
