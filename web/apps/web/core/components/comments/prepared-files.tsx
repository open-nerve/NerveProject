/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import type { CommentPreparations } from "@/store/asset/comment-preparations";
import type { UploadSession } from "@/store/asset/upload-session";
import { errorMessageKey } from "@/lib/error-messages";
import { AttachmentPicker } from "@/components/assets/attachments/picker";
import { AttachmentList } from "@/components/assets/attachments/list";
export const PreparedCommentFiles = observer(function PreparedCommentFiles({
  files,
  upload,
}: {
  files: CommentPreparations;
  upload: UploadSession | undefined;
}) {
  const { t } = useTranslation();
  if (!files.current) return null;
  return (
    <section aria-label={t("content_files.comment.files")} className="space-y-2">
      {upload && files.ready && files.ids.length < 100 && (
        <AttachmentPicker upload={upload} onUploaded={async (asset) => files.add(asset)} />
      )}
      {upload && !["idle", "cancelled", "ready"].includes(upload.phase) && (
        <button type="button" onClick={() => upload.cancel()}>
          {t("content_files.comment.cancel_upload")}
        </button>
      )}
      {files.loading && <p role="status">{t("content_files.reading")}</p>}
      {files.error !== undefined && <p role="alert">{t(errorMessageKey(files.error))}</p>}
      {files.unresolved.map((id, index) => (
        <div key={id}>
          {t("content_files.comment.unverified")} {index + 1}{" "}
          <button type="button" onClick={() => files.remove(id)}>
            {t("content_files.comment.remove_selection")}
          </button>
        </div>
      ))}
      {(files.unresolved.length > 0 || files.error !== undefined) && !files.loading && (
        <button type="button" onClick={() => void files.load().catch(() => undefined)}>
          {t("content_files.refresh")}
        </button>
      )}
      <AttachmentList
        rows={files.rows}
        remove={(row) => {
          files.remove(row.id);
          if (upload?.readyAsset?.id === row.id) upload.cancel();
        }}
      />
      {files.ids.length === 100 && <p role="status">{t("content_files.comment.limit")}</p>}
    </section>
  );
});
