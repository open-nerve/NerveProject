/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { ImageOperation, ImageTransferPhase } from "@/types";
import { isImageID } from "../input";
import { closeHistory } from "@tiptap/pm/history";
import { ECustomImageStatus } from "../types";
import type { CustomImageNodeViewProps } from "./node-view";
export function useImageTransfer(props: CustomImageNodeViewProps) {
  const { editor, extension, node } = props;
  const images = extension.options.images;
  const id = node.attrs.id ?? "";
  const src = node.attrs.src;
  const latest = useRef(props);
  latest.current = props;
  const active = useRef<AbortController | undefined>(undefined);
  const mounted = useRef(false);
  const [phase, setPhase] = useState<ImageTransferPhase>();
  const [error, setError] = useState<string>();
  const [failed, setFailed] = useState(false);
  const [copied, setCopied] = useState(false);
  const pending = extension.storage.fileMap;
  const current = useCallback(
    (controller: AbortController) => {
      if (
        !mounted.current ||
        editor.isDestroyed ||
        !editor.isEditable ||
        active.current !== controller ||
        controller.signal.aborted ||
        !images ||
        images.signal.aborted
      )
        return false;
      const pos = latest.current.getPos();
      const currentNode = pos === undefined ? undefined : editor.state.doc.nodeAt(pos);
      return (
        !!currentNode &&
        currentNode.type.name === node.type.name &&
        currentNode.attrs.id === id &&
        currentNode.attrs.src === src
      );
    },
    [editor, id, images, node.type.name, src]
  );
  const run = useCallback(
    (mode: "upload" | "retry" | "duplicate", file?: File) => {
      if (!mounted.current || !images?.write || images.signal.aborted || !editor.isEditable || !id) return;
      active.current?.abort();
      const controller = new AbortController();
      active.current = controller;
      setCopied(mode === "duplicate");
      setFailed(false);
      setError(undefined);
      setPhase("queued");
      const options: ImageOperation = {
        signal: controller.signal,
        onPhase: (next) => {
          if (current(controller)) setPhase(next);
        },
      };
      const source = pending.get(id)?.sourceID;
      const operation =
        mode === "upload" && file
          ? () => images.write!.upload(id, file, options)
          : mode === "retry"
            ? () => images.write!.retryUpload(id, options)
            : source
              ? () => images.write!.duplicate(id, source, options)
              : undefined;
      if (!operation) {
        setPhase(undefined);
        return;
      }
      void Promise.resolve()
        .then(() => {
          if (!current(controller)) throw new Error("Cancelled");
          return operation();
        })
        .then((assetID) => {
          if (!current(controller)) return;
          if (!isImageID(assetID)) throw new Error("Invalid image identity");
          setPhase(undefined);
          setError(undefined);
          setFailed(false);
          pending.delete(id);
          if (current(controller)) {
            const pos = latest.current.getPos()!;
            const currentNode = editor.state.doc.nodeAt(pos)!;
            extension.storage.completed.set(id, assetID);
            editor.view.dispatch(
              editor.state.tr
                .setNodeMarkup(pos, undefined, {
                  ...currentNode.attrs,
                  src: assetID,
                  status: ECustomImageStatus.UPLOADED,
                })
                .setMeta("addToHistory", false)
            );
            if (!editor.isDestroyed) editor.view.dispatch(closeHistory(editor.state.tr));
          }
          return undefined;
        })
        .catch((cause: unknown) => {
          if (current(controller)) {
            setPhase(undefined);
            setFailed(true);
            setError(cause instanceof Error ? cause.message : undefined);
          }
        });
    },
    [current, editor, extension, id, images, pending]
  );
  useLayoutEffect(() => {
    mounted.current = true;
    let alive = true;
    const stop = () => {
      active.current?.abort();
      images?.write?.cancel(id);
    };
    images?.signal.addEventListener("abort", stop, { once: true });
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
      mounted.current = false;
      images?.signal.removeEventListener("abort", stop);
      stop();
      const entry = pending.get(id);
      if (entry && active.current) {
        entry.autoStart = false;
        entry.file = undefined;
      }
    };
  }, [id, images, pending, run]);
  const cancel = () => {
    active.current?.abort();
    images?.write?.cancel(id);
    setPhase(undefined);
    setFailed(false);
    setError(undefined);
  };
  return {
    phase,
    error,
    failed,
    copied,
    upload: (file: File) => run("upload", file),
    retry: () => run(copied ? "duplicate" : "retry"),
    cancel,
  };
}
