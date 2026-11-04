/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { FullScreenOutline } from "@makeplane/propel/icons";
import { useContext, useEffect, useState } from "react";
// nerve imports
import type { Editor } from "@tiptap/core";
import { EditorLabelsContext } from "@/components/editor-labels";
import { Tooltip } from "@nerve/propel/tooltip";
// local imports
import { ImageFullScreenModal } from "./modal";

type Props = {
  editor: Editor;
  image: {
    aspectRatio: number;
    download: (signal: AbortSignal) => Promise<void>;
    src: string;
    width: string;
  };
  isTouchDevice: boolean;
  toggleToolbarViewStatus: (val: boolean) => void;
};

export function ImageFullScreenActionRoot(props: Props) {
  const { image: labels } = useContext(EditorLabelsContext);
  const { editor, image, isTouchDevice, toggleToolbarViewStatus } = props;
  // states
  const [isFullScreenEnabled, setIsFullScreenEnabled] = useState(false);
  // derived values
  const { download, src, width, aspectRatio } = image;

  useEffect(() => {
    toggleToolbarViewStatus(isFullScreenEnabled);
  }, [isFullScreenEnabled, toggleToolbarViewStatus]);

  useEffect(() => {
    if (!isFullScreenEnabled) return;
    editor.commands.addActiveDropbarExtension("image-viewer");
    return () => {
      if (!editor.isDestroyed) editor.commands.removeActiveDropbarExtension("image-viewer");
    };
  }, [editor, isFullScreenEnabled]);

  return (
    <>
      <ImageFullScreenModal
        aspectRatio={aspectRatio}
        download={download}
        isFullScreenEnabled={isFullScreenEnabled}
        src={src}
        width={width}
        toggleFullScreenMode={setIsFullScreenEnabled}
      />
      <Tooltip tooltipContent={labels.fullScreen} disabled={isTouchDevice}>
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            setIsFullScreenEnabled(true);
          }}
          className="grid h-full flex-shrink-0 place-items-center text-on-color/60 transition-colors hover:text-on-color"
          aria-label={labels.fullScreen}
        >
          <FullScreenOutline className="size-3" />
        </button>
      </Tooltip>
    </>
  );
}
