/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { File, Trash2, Download } from "lucide-react";
import type { Attachment } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
export type AttachmentRow = Pick<Attachment, "id" | "name" | "size" | "content_type">;
/** Plain metadata also presents selected new-comment preparations; it performs no implicit reads. */
export function AttachmentItem({
  row,
  download,
  remove,
}: {
  row: AttachmentRow;
  download?(id: string): Promise<void>;
  remove?(row: AttachmentRow): void;
}) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const pending = useRef(false);
  const life = useRef(false);
  const alert = useRef<HTMLParagraphElement>(null);
  useEffect(() => {
    life.current = true;
    return () => {
      life.current = false;
    };
  }, []);
  useLayoutEffect(() => {
    if (failure) alert.current?.focus({ preventScroll: true });
  }, [failure]);
  const save = async () => {
    if (!download || pending.current) return;
    pending.current = true;
    setBusy(true);
    setFailure(undefined);
    try {
      await download(row.id);
    } catch (error) {
      if (life.current) setFailure(errorMessageKey(error));
    } finally {
      if (life.current) {
        pending.current = false;
        setBusy(false);
      }
    }
  };
  return (
    <li className="rounded-md border border-subtle p-3">
      <div className="flex items-center gap-3">
        <File aria-hidden="true" className="size-4 shrink-0 text-secondary" />
        <span className="min-w-0 flex-1 text-13 break-words">{row.name}</span>
        <span className="shrink-0 text-12 text-secondary">{row.size} B</span>
        {download && (
          <button
            type="button"
            className="rounded p-1 hover:bg-layer-1"
            aria-label={`${t("content_files.download")} ${row.name}`}
            disabled={busy}
            onClick={() => {
              void save();
            }}
          >
            <Download aria-hidden="true" className="size-4" />
          </button>
        )}
        {remove && (
          <button
            type="button"
            className="rounded p-1 hover:bg-layer-1"
            aria-label={`${t("content_files.remove")} ${row.name}`}
            onClick={() => remove(row)}
          >
            <Trash2 aria-hidden="true" className="size-4" />
          </button>
        )}
      </div>
      {busy && (
        <p aria-live="polite" className="text-12 text-secondary">
          {t("content_files.downloading")}
        </p>
      )}
      {failure && (
        <p ref={alert} tabIndex={-1} role="alert" className="text-13 text-danger-primary">
          {t(failure)}
        </p>
      )}
    </li>
  );
}
