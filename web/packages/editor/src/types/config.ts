/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type ImageTransferPhase = "queued" | "preparing" | "uploading" | "confirming";
export type ImageOperation = { signal: AbortSignal; onPhase: (phase: ImageTransferPhase) => void };
export type EditorImageSnapshot =
  | { phase: "loading" | "unavailable" }
  | { phase: "error"; message: string }
  | { phase: "ready"; url: string; expiresAt: string };
export type EditorImageRead = {
  getSnapshot: () => EditorImageSnapshot;
  subscribe: (changed: () => void) => () => void;
  loaded: () => void;
  failed: () => void;
  retry: () => void;
  dispose: () => void;
};
export type EditorImages = {
  signal: AbortSignal;
  read: (id: string) => EditorImageRead;
  download: (id: string, signal: AbortSignal) => Promise<void>;
  write?: {
    upload: (nodeID: string, file: File, operation: ImageOperation) => Promise<string>;
    retryUpload: (nodeID: string, operation: ImageOperation) => Promise<string>;
    duplicate: (nodeID: string, sourceID: string, operation: ImageOperation) => Promise<string>;
    cancel: (nodeID: string) => void;
    maxBytes: number;
  };
};
type TEditorFontSize = "small-font" | "large-font" | "mobile-font";

type TEditorLineSpacing = "regular" | "small" | "mobile-regular";

export type TDisplayConfig = {
  fontSize?: TEditorFontSize;
  lineSpacing?: TEditorLineSpacing;
};

export type EditorColor = "gray" | "peach" | "pink" | "orange" | "green" | "light-blue" | "dark-blue" | "purple";

/** Consumer-supplied labels, shared with React NodeViews through the editor's context. */
export type EditorLabels = {
  image: Record<
    | "select"
    | "cancel"
    | "retry"
    | "unavailable"
    | "loading"
    | "copyUnknown"
    | "cancelled"
    | "invalid"
    | "size"
    | "name"
    | "tooMany"
    | "queued"
    | "preparing"
    | "uploading"
    | "confirming"
    | "download"
    | "downloading"
    | "downloadFailed"
    | "fullScreen"
    | "close"
    | "zoomIn"
    | "zoomOut"
    | "align"
    | "left"
    | "center"
    | "right",
    string
  >;
  continueWriting: string;
  resizeImage: string;
  tableRowOptions: string;
  tableColumnOptions: string;
  tableColors: Record<EditorColor | "none", string>;
};
