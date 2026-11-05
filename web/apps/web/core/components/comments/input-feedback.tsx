/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useRef, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import { errorMessageKey } from "@/lib/error-messages";
import type { RetainedForm } from "../issues/detail-input-state";
export function CommentInputFeedback({
  value,
  current,
  verify,
  verified,
  allowResend,
}: {
  value: RetainedForm;
  current(): boolean;
  verify(): Promise<unknown>;
  verified(): void;
  allowResend(): void;
}) {
  const { t } = useTranslation();
  const reading = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const read = async () => {
    if (reading.current || !current()) return;
    reading.current = true;
    setBusy(true);
    setError(undefined);
    try {
      await verify();
      if (current()) verified();
    } catch (reason) {
      if (current()) setError(reason);
    } finally {
      reading.current = false;
      if (current()) setBusy(false);
    }
  };
  return (
    <>
      {value.error !== undefined && <p role="alert">{t(errorMessageKey(value.error))}</p>}
      {value.unknown && (
        <div role="status" className="space-x-2">
          <p>{t("content_files.comment.unknown")}</p>
          <button type="button" disabled={busy} onClick={() => void read()}>
            {t("content_files.comment.verify")}
          </button>
          {value.verified && (
            <button type="button" disabled={busy} onClick={allowResend}>
              {t("content_files.comment.allow_resend")}
            </button>
          )}
        </div>
      )}
      {error !== undefined && <p role="alert">{t(errorMessageKey(error))}</p>}
    </>
  );
}
