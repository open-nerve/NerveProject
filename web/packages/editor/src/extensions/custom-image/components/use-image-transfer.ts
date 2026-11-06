/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useSyncExternalStore } from "react";
import type { ImageOperation } from "@/types";
import type { CustomImageNodeViewProps } from "./node-view";
/** This node's transfer, which the editor keeps across node views; the view only starts, shows and cancels it. */
export function useImageTransfer(props: CustomImageNodeViewProps) {
  const { editor, extension, node } = props;
  const images = extension.options.images;
  const { fileMap: pending, transfers } = extension.storage;
  const id = node.attrs.id ?? "";
  const state = useSyncExternalStore(
    useCallback((changed: () => void) => transfers.subscribe(id, changed), [transfers, id]),
    () => transfers.state(id)
  );
  const run = useCallback(
    (mode: "upload" | "retry" | "duplicate", file?: File) => {
      const write = images?.write;
      if (!write || images.signal.aborted || !editor.isEditable || !id) return;
      const source = pending.get(id)?.sourceID;
      const operation =
        mode === "upload" && file
          ? (options: ImageOperation) => write.upload(id, file, options)
          : mode === "retry"
            ? (options: ImageOperation) => write.retryUpload(id, options)
            : source
              ? (options: ImageOperation) => write.duplicate(id, source, options)
              : undefined;
      if (operation) transfers.start(editor, id, mode === "duplicate", operation);
    },
    [editor, id, images, pending, transfers]
  );
  useLayoutEffect(() => {
    let alive = true;
    queueMicrotask(() => {
      if (!alive) return;
      const entry = pending.get(id);
      if (!entry?.autoStart) return;
      entry.autoStart = false;
      if (entry.file) {
        const file = entry.file;
        entry.file = undefined;
        run("upload", file);
      } else if (entry.sourceID) run("duplicate");
    });
    return () => {
      alive = false;
      transfers.release(editor, id);
    };
  }, [editor, id, pending, run, transfers]);
  return {
    ...state,
    upload: (file: File) => run("upload", file),
    retry: () => run(state.copied ? "duplicate" : "retry"),
    cancel: () => transfers.cancel(id),
  };
}
