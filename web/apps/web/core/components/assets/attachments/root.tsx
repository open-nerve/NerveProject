/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useEffect, useRef, useState } from "react";
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
}: {
  view: ContentView;
  resource: AttachmentResource;
  readOnly?: boolean;
  onDialogOpened?(): () => void;
}) {
  const { t } = useTranslation();
  const root = useContext(StoreContext);
  const upload = useContentUpload(readOnly ? undefined : view, "attachment");
  const refresh = useRef<HTMLButtonElement>(null);
  const [selection, setSelection] = useState<{ resource: AttachmentResource; row: AttachmentRow }>();
  useEffect(() => {
    void resource.load().catch(() => undefined);
  }, [resource]);
  if (!view.current) return null;
  const selected = selection?.resource === resource ? selection.row : undefined;
  return (
    <div className="space-y-3">
      {upload && <AttachmentPicker upload={upload} onUploaded={(asset) => resource.afterUpload(asset)} />}
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
          remove={(id) => resource.remove(id)}
          verify={() => resource.retryRead()}
          onClose={() => setSelection(undefined)}
        />
      )}
    </div>
  );
});
