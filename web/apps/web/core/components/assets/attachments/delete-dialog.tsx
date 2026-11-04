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
  verify?(): Promise<void>;
  onClose(): void;
};
export const DeleteAttachmentDialog = observer(function DeleteAttachmentDialog({
  view,
  file,
  discard = false,
  uncertain = false,
  remove,
  verify,
  onClose,
}: DeleteAttachmentProps) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [verified, setVerified] = useState(false);
  const pending = useRef(false);
  const mounted = useRef(false);
  const alert = useRef<HTMLParagraphElement>(null);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useLayoutEffect(() => {
    if (error && view.current) alert.current?.focus({ preventScroll: true });
  }, [error, view]);
  const perform = async (readOnly: boolean) => {
    if (pending.current || !view.current || (!readOnly && uncertain && !verified)) return;
    pending.current = true;
    setBusy(true);
    setError(undefined);
    try {
      if (readOnly) await verify?.();
      else await remove(file.id);
      if (mounted.current && view.current) {
        if (readOnly) setVerified(true);
        else onClose();
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
        <p className="text-13 text-secondary">
          {t(discard ? "content_files.preparations.confirm_warning" : "content_files.delete_warning")}
        </p>
        {uncertain && <p className="text-13 text-secondary">{t("content_files.unknown_delete")}</p>}
        {error && (
          <p ref={alert} role="alert" tabIndex={-1} className="text-13 text-danger-primary">
            {t(error)}
          </p>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <Button type="button" variant="secondary" disabled={busy} onClick={close}>
            {t("cancel")}
          </Button>
          {uncertain && verify && (
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
        </div>
      </div>
    </ModalCore>
  );
});
