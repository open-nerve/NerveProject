/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { useState } from "react";
// nerve imports
import { cn } from "@nerve/utils";
// local imports
import type { TCustomImageAlignment } from "../../types";
import { ImageAlignmentAction } from "./alignment";
import { ImageDownloadAction } from "./download";
import { ImageFullScreenActionRoot } from "./full-screen";

type Props = {
  alignment: TCustomImageAlignment;
  editor: Editor;
  aspectRatio: number;
  download: (signal: AbortSignal) => Promise<void>;
  handleAlignmentChange: (alignment: TCustomImageAlignment) => void;
  isTouchDevice: boolean;
  src: string;
  width: string;
};

export function ImageToolbarRoot(props: Props) {
  const { alignment, editor, download, handleAlignmentChange, isTouchDevice } = props;
  // states
  const [shouldShowToolbar, setShouldShowToolbar] = useState(false);
  // derived values
  const isEditable = editor.isEditable;

  return (
    <>
      <div
        className={cn(
          "pointer-events-none absolute top-1 right-1 z-20 flex h-7 items-center gap-2 rounded-sm bg-black/80 px-2 opacity-0 transition-opacity group-hover/image-component:pointer-events-auto group-hover/image-component:opacity-100 focus-within:pointer-events-auto focus-within:opacity-100",
          {
            "pointer-events-auto opacity-100": shouldShowToolbar,
          }
        )}
      >
        <ImageDownloadAction download={download} />
        {isEditable && (
          <ImageAlignmentAction
            activeAlignment={alignment}
            handleChange={handleAlignmentChange}
            isTouchDevice={isTouchDevice}
            toggleToolbarViewStatus={setShouldShowToolbar}
          />
        )}
        <ImageFullScreenActionRoot
          editor={editor}
          image={props}
          isTouchDevice={isTouchDevice}
          toggleToolbarViewStatus={setShouldShowToolbar}
        />
      </div>
    </>
  );
}
