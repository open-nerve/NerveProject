/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Node } from "@tiptap/core";
// types
import type { EditorImages } from "@/types";
import type { ImageInputError } from "./input";
import type { ImageTransfers } from "./transfers";

export enum ECustomImageAttributeNames {
  ID = "id",
  WIDTH = "width",
  HEIGHT = "height",
  ASPECT_RATIO = "aspectRatio",
  SOURCE = "src",
  ALIGNMENT = "alignment",
  STATUS = "status",
}

export type Pixel = `${number}px`;

type PixelAttribute<TDefault> = Pixel | TDefault;

export type TCustomImageSize = {
  width: PixelAttribute<"35%">;
  height: PixelAttribute<"auto">;
  aspectRatio: number | null;
};

export type TCustomImageAlignment = "left" | "center" | "right";

export enum ECustomImageStatus {
  PENDING = "pending",
  UPLOADED = "uploaded",
}

export type TCustomImageAttributes = {
  [ECustomImageAttributeNames.ID]: string | null;
  [ECustomImageAttributeNames.WIDTH]: PixelAttribute<"35%" | number> | null;
  [ECustomImageAttributeNames.HEIGHT]: PixelAttribute<"auto" | number> | null;
  [ECustomImageAttributeNames.ASPECT_RATIO]: number | null;
  [ECustomImageAttributeNames.SOURCE]: string | null;
  [ECustomImageAttributeNames.ALIGNMENT]: TCustomImageAlignment;
  [ECustomImageAttributeNames.STATUS]: ECustomImageStatus;
};

type ImagePending = { file?: File; sourceID?: string; autoStart: boolean; openPicker?: boolean };

export type InsertImageComponentProps = {
  file?: File;
  pos?: number;
  event: "insert" | "drop";
};

export type CustomImageExtensionOptions = { images?: EditorImages };
export type CustomImageExtensionStorage = {
  fileMap: Map<string, ImagePending>;
  completed: Map<string, string>;
  /** Transfers by node id, outliving the node views that show them. */
  transfers: ImageTransfers;
  images?: EditorImages;
  inputError?: ImageInputError;
};

export type CustomImageExtensionType = Node<CustomImageExtensionOptions, CustomImageExtensionStorage>;
