/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { NodeSelection } from "@tiptap/pm/state";
import { useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from "react";
import { cn, isComposing } from "@nerve/utils";
import { EditorLabelsContext } from "@/components/editor-labels";
import { ensurePixelString, getImageBlockId } from "../utils";
import type { TCustomImageSize } from "../types";
import type { CustomImageNodeViewProps } from "./node-view";
import { ImageToolbarRoot } from "./toolbar";
type Props = CustomImageNodeViewProps & {
  src: string;
  loaded: () => void;
  failed: () => void;
  download: (signal: AbortSignal) => Promise<void>;
};
const MIN_SIZE = 100;
export function CustomImageBlock({
  editor,
  node,
  getPos,
  selected,
  updateAttributes,
  src,
  loaded,
  failed,
  download,
}: Props) {
  const labels = useContext(EditorLabelsContext);
  const { width, height, aspectRatio, alignment } = node.attrs;
  const [size, setSize] = useState<TCustomImageSize>({
    width: ensurePixelString(width, "35%") ?? "35%",
    height: ensurePixelString(height, "auto") ?? "auto",
    aspectRatio,
  });
  const container = useRef<HTMLDivElement>(null);
  const rect = useRef<DOMRect | null>(null);
  const [resizing, setResizing] = useState(false);
  const current = useCallback(() => {
    if (editor.isDestroyed || !editor.isEditable) return false;
    const pos = getPos();
    const actual = pos === undefined ? undefined : editor.state.doc.nodeAt(pos);
    return actual?.attrs.id === node.attrs.id && actual?.attrs.src === node.attrs.src;
  }, [editor, getPos, node.attrs.id, node.attrs.src]);
  useLayoutEffect(
    () =>
      setSize({
        width: ensurePixelString(width, "35%") ?? "35%",
        height: ensurePixelString(height, "auto") ?? "auto",
        aspectRatio,
      }),
    [width, height, aspectRatio]
  );
  const resize = useCallback(
    (event: MouseEvent | TouchEvent) => {
      if (!rect.current || !size.aspectRatio) return;
      const x = "touches" in event ? event.touches[0]!.clientX : event.clientX;
      const nextWidth = Math.max(MIN_SIZE, alignment === "right" ? rect.current.right - x : x - rect.current.left);
      setSize((previous) => ({ ...previous, width: `${nextWidth}px`, height: `${nextWidth / size.aspectRatio!}px` }));
    },
    [alignment, size.aspectRatio]
  );
  const end = useCallback(() => {
    setResizing(false);
    if (current()) updateAttributes(size);
  }, [current, size, updateAttributes]);
  useEffect(() => {
    if (!resizing) return;
    window.addEventListener("mousemove", resize);
    window.addEventListener("touchmove", resize);
    window.addEventListener("mouseup", end);
    window.addEventListener("touchend", end);
    return () => {
      window.removeEventListener("mousemove", resize);
      window.removeEventListener("touchmove", resize);
      window.removeEventListener("mouseup", end);
      window.removeEventListener("touchend", end);
    };
  }, [resizing, resize, end]);
  const byKeyboard = (amount: number) => {
    if (!size.aspectRatio || !current()) return;
    const nextWidth = Math.max(MIN_SIZE, parseFloat(size.width) + amount);
    const next: TCustomImageSize = { ...size, width: `${nextWidth}px`, height: `${nextWidth / size.aspectRatio}px` };
    setSize(next);
    updateAttributes(next);
  };
  return (
    <div
      id={node.attrs.id ? getImageBlockId(node.attrs.id) : undefined}
      className={cn("w-fit max-w-full", {
        "ml-[50%] -translate-x-1/2": alignment === "center",
        "ml-[100%] -translate-x-full": alignment === "right",
      })}
    >
      <div
        ref={container}
        className="group/image-component relative inline-block max-w-full"
        style={{ width: size.width, aspectRatio: size.aspectRatio ?? undefined }}
      >
        <img
          className="image-component block rounded-md"
          alt=""
          src={src}
          referrerPolicy="no-referrer"
          style={{ width: size.width, aspectRatio: size.aspectRatio ?? undefined }}
          onLoad={(event) => {
            loaded();
            const img = event.currentTarget;
            if (!img.naturalWidth || !img.naturalHeight) return;
            const ratio = img.naturalWidth / img.naturalHeight;
            const localWidth =
              width === "35%"
                ? Math.max(MIN_SIZE, (container.current?.closest(".editor-container")?.clientWidth ?? 0) * 0.35)
                : parseFloat(size.width);
            // Intrinsic size only affects this view. Resolving/loading a capability must never save the document.
            setSize({ width: `${localWidth}px`, height: `${localWidth / ratio}px`, aspectRatio: ratio });
          }}
          onError={failed}
          onMouseDown={(event) => {
            event.stopPropagation();
            const pos = getPos();
            if (!editor.isDestroyed && pos !== undefined)
              editor.view.dispatch(editor.state.tr.setSelection(NodeSelection.create(editor.state.doc, pos)));
          }}
        />
        <ImageToolbarRoot
          editor={editor}
          alignment={alignment ?? "left"}
          aspectRatio={size.aspectRatio ?? 1}
          width={size.width}
          src={src}
          download={download}
          handleAlignmentChange={(nextAlignment) => {
            if (current()) updateAttributes({ alignment: nextAlignment });
          }}
          isTouchDevice={!!editor.storage.utility.isTouchDevice}
        />
        {selected && <div className="pointer-events-none absolute inset-0 bg-accent-primary/30" />}
        {editor.isEditable && (
          <button
            type="button"
            aria-label={labels.resizeImage}
            className="absolute right-0 bottom-0 size-4 rounded-full bg-accent-primary focus-visible:outline-2"
            onMouseDown={(event) => {
              event.preventDefault();
              event.stopPropagation();
              rect.current = container.current?.getBoundingClientRect() ?? null;
              setResizing(true);
            }}
            onTouchStart={(event) => {
              event.preventDefault();
              rect.current = container.current?.getBoundingClientRect() ?? null;
              setResizing(true);
            }}
            onClick={(event) => {
              if (event.detail === 0) byKeyboard(10);
            }}
            onKeyDown={(event) => {
              if (isComposing(event.nativeEvent) || !["ArrowLeft", "ArrowRight"].includes(event.key)) return;
              event.preventDefault();
              event.stopPropagation();
              byKeyboard(event.key === "ArrowRight" ? 10 : -10);
            }}
          />
        )}
      </div>
    </div>
  );
}
