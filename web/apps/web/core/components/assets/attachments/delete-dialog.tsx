/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import type { ContentView } from "@/store/asset/content-view";
import { errorMessageKey } from "@/lib/error-messages";
export type DeleteAttachmentProps = {
  view: ContentView;
  file: { id: string; name: string };
  discard?: boolean;
  uncertain?: boolean;
  remove(id: string): Promise<void>;
  /** Reads the latest state; resolves to the uncertain IDs that are still visible. */
  verify?(): Promise<ReadonlySet<string>>;
  onClose(): void;
  onDialogOpened?(): () => void;
};
export const DeleteAttachmentDialog = observer(function DeleteAttachmentDialog({
  view,
  file,
  discard = false,
  uncertain = false,
  remove,
  verify,
  onClose,
  onDialogOpened,
}: DeleteAttachmentProps) {
  const { t } = useTranslation();
  useLayoutEffect(() => onDialogOpened?.(), [onDialogOpened]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [verified, setVerified] = useState(false);
  // The read found the file gone: nothing is left to delete, so the dialog only explains and closes.
  const [gone, setGone] = useState(false);
  const pending = useRef(false);
  const mounted = useRef(false);
  const alert = useRef<HTMLParagraphElement>(null);
  const notice = useRef<HTMLParagraphElement>(null);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useLayoutEffect(() => {
    if (error && view.current) alert.current?.focus({ preventScroll: true });
  }, [error, view]);
  useLayoutEffect(() => {
    if (gone && view.current) notice.current?.focus({ preventScroll: true });
  }, [gone, view]);
  const perform = async (readOnly: boolean) => {
    if (pending.current || gone || !view.current || (!readOnly && uncertain && !verified)) return;
    pending.current = true;
    setBusy(true);
    setError(undefined);
    try {
      if (readOnly) {
        const visible = await verify?.();
        if (mounted.current && view.current) {
          if (visible?.has(file.id)) setVerified(true);
          else setGone(true);
        }
      } else {
        await remove(file.id);
        if (mounted.current && view.current) onClose();
      }
    } catch (reason) {
      if (mounted.current && view.current) {
        setVerified(false);
        setError(errorMessageKey(reason));
      }
    } finally {
      if (mounted.current && view.current) {
        pending.current = false;
        setBusy(false);
      }
    }
  };
  const close = () => {
    if (!pending.current) onClose();
  };
  return (
    <ModalCore isOpen={view.current} handleClose={busy ? undefined : close} width={EModalWidth.MD}>
      <div className="space-y-4 p-6">
        <ModalTitle>{t(discard ? "content_files.preparations.delete_title" : "content_files.delete_title")}</ModalTitle>
        <p className="text-13 break-words">{file.name}</p>
        {gone ? (
          <p ref={notice} tabIndex={-1} className="text-13 text-secondary">
            {t("content_files.delete_gone")}
          </p>
        ) : (
          <>
            <p className="text-13 text-secondary">
              {t(discard ? "content_files.preparations.confirm_warning" : "content_files.delete_warning")}
            </p>
            {uncertain && <p className="text-13 text-secondary">{t("content_files.unknown_delete")}</p>}
          </>
        )}
        {error && (
          <p ref={alert} role="alert" tabIndex={-1} className="text-13 text-danger-primary">
            {t(error)}
          </p>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <Button type="button" variant="secondary" disabled={busy} onClick={close}>
            {t(gone ? "close" : "cancel")}
          </Button>
          {!gone && uncertain && verify && (
            <Button
              type="button"
              variant="secondary"
              disabled={busy}
              onClick={() => {
                void perform(true);
              }}
            >
              {t("content_files.verify")}
            </Button>
          )}
          {!gone && (
            <Button
              type="button"
              variant="error-fill"
              disabled={busy || (uncertain && !verified)}
              onClick={() => {
                void perform(false);
              }}
            >
              {t(discard ? "content_files.confirm_discard" : "content_files.confirm_delete")}
            </Button>
          )}
        </div>
      </div>
    </ModalCore>
  );
});
