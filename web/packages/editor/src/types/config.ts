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
export type TFileHandler = {
  assetsUploadStatus: Record<string, number>; // blockId => progress percentage
  delete: (assetSrc: string) => Promise<void>;
  getAssetDownloadSrc: (path: string) => Promise<string>;
  getAssetSrc: (path: string) => Promise<string>;
  restore: (assetSrc: string) => Promise<void>;
  upload: (blockId: string, file: File) => Promise<string>;
  duplicate: (assetId: string) => Promise<string>;
  validation: {
    /**
     * @description max file size in bytes
     * @example enter 5242880(5 * 1024 * 1024) for 5MB
     */
    maxFileSize: number;
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
  continueWriting: string;
  resizeImage: string;
  tableRowOptions: string;
  tableColumnOptions: string;
  tableColors: Record<EditorColor | "none", string>;
};
