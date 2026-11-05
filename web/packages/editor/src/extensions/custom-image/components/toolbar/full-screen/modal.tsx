/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Dialog, DialogPanel } from "@headlessui/react";
import { useContext, useEffect, useRef, useState } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
import { ImageDownloadAction } from "../download";
type Props = {
  aspectRatio: number;
  download: (signal: AbortSignal) => Promise<void>;
  isFullScreenEnabled: boolean;
  src: string;
  toggleFullScreenMode: (value: boolean) => void;
  width: string;
};
export function ImageFullScreenModal({
  aspectRatio,
  download,
  isFullScreenEnabled,
  src,
  toggleFullScreenMode,
  width,
}: Props) {
  const { image: labels } = useContext(EditorLabelsContext);
  const [zoom, setZoom] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const drag = useRef<{ x: number; y: number; left: number; top: number } | undefined>(undefined);
  const [panel, setPanel] = useState<HTMLElement | null>(null);
  const close = () => {
    toggleFullScreenMode(false);
    setZoom(1);
    setOffset({ x: 0, y: 0 });
    drag.current = undefined;
  };
  const step = (direction: number) => {
    const next = Math.max(0.5, Math.min(2, (direction > 0 ? Math.floor(zoom * 2) + 1 : Math.ceil(zoom * 2) - 1) / 2));
    if (next === 1) setOffset({ x: 0, y: 0 });
    setZoom(next);
  };
  useEffect(() => {
    const node = panel;
    if (!isFullScreenEnabled || !node) return;
    const wheel = (event: WheelEvent) => {
      if (!event.ctrlKey && !event.metaKey) return;
      event.preventDefault();
      setZoom((value) => Math.max(0.5, Math.min(2, value * (1 - event.deltaY * 0.05))));
    };
    node.addEventListener("wheel", wheel, { passive: false });
    return () => node.removeEventListener("wheel", wheel);
  }, [isFullScreenEnabled, panel]);
  const baseWidth =
    typeof window === "undefined"
      ? parseFloat(width)
      : Math.min(window.innerWidth * 0.9, window.innerHeight * 0.75 * aspectRatio);
  return (
    <Dialog open={isFullScreenEnabled} onClose={close} className="relative z-[100]" aria-label={labels.fullScreen}>
      <div className="fixed inset-0 bg-black/90" aria-hidden="true" />
      <div className="fixed inset-0 grid place-items-center overflow-auto p-6">
        <DialogPanel
          ref={setPanel}
          className="flex max-w-full flex-col items-center gap-4"
          onKeyDown={(event) => {
            if (!["+", "=", "-"].includes(event.key)) return;
            event.preventDefault();
            event.stopPropagation();
            step(event.key === "-" ? -1 : 1);
          }}
        >
          <button type="button" className="fixed top-10 right-10 z-10 text-white" onClick={close}>
            {labels.close}
          </button>
          <img
            alt=""
            src={src}
            referrerPolicy="no-referrer"
            draggable={false}
            className="read-only-image rounded-lg"
            style={{
              width: Number.isFinite(baseWidth) ? baseWidth : parseFloat(width),
              maxWidth: "none",
              maxHeight: "none",
              objectFit: "contain",
              aspectRatio,
              transform: `translate(${offset.x}px, ${offset.y}px) scale(${zoom})`,
              touchAction: "none",
            }}
            onPointerDown={(event) => {
              if (event.button !== 0) return;
              event.preventDefault();
              drag.current = { x: event.clientX, y: event.clientY, left: offset.x, top: offset.y };
              event.currentTarget.setPointerCapture(event.pointerId);
            }}
            onPointerMove={(event) => {
              const from = drag.current;
              if (from) setOffset({ x: from.left + event.clientX - from.x, y: from.top + event.clientY - from.y });
            }}
            onPointerUp={() => {
              drag.current = undefined;
            }}
            onPointerCancel={() => {
              drag.current = undefined;
            }}
          />
          <div className="fixed bottom-10 left-1/2 z-10 flex -translate-x-1/2 gap-4 rounded bg-black p-3 text-white">
            <button type="button" disabled={zoom <= 0.5} onClick={() => step(-1)}>
              {labels.zoomOut}
            </button>
            <span>{Math.round(zoom * 100)}%</span>
            <button type="button" disabled={zoom >= 2} onClick={() => step(1)}>
              {labels.zoomIn}
            </button>
            <ImageDownloadAction download={download} />
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  );
}
