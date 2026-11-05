/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { DownloadOutline } from "@makeplane/propel/icons";
import { useContext, useEffect, useRef, useState } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
export function ImageDownloadAction({ download }: { download: (signal: AbortSignal) => Promise<void> }) {
  const { image: labels } = useContext(EditorLabelsContext);
  const active = useRef<AbortController | undefined>(undefined);
  const [busy, setBusy] = useState(false),
    [failed, setFailed] = useState(false);
  useEffect(
    () => () => {
      active.current?.abort();
      active.current = undefined;
    },
    []
  );
  const start = () => {
    if (active.current) return;
    const controller = new AbortController();
    active.current = controller;
    setBusy(true);
    setFailed(false);
    void download(controller.signal)
      .catch(() => {
        if (active.current === controller && !controller.signal.aborted) setFailed(true);
      })
      .finally(() => {
        if (active.current === controller) {
          active.current = undefined;
          setBusy(false);
        }
      });
  };
  return (
    <>
      <button
        type="button"
        onClick={start}
        disabled={busy}
        aria-label={busy ? labels.downloading : labels.download}
        className="grid h-full shrink-0 place-items-center text-white"
      >
        <DownloadOutline className="size-4" />
      </button>
      {busy && (
        <button
          type="button"
          onClick={() => {
            active.current?.abort();
            active.current = undefined;
            setBusy(false);
          }}
        >
          {labels.cancel}
        </button>
      )}
      {failed && <span role="alert">{labels.downloadFailed}</span>}
    </>
  );
}
