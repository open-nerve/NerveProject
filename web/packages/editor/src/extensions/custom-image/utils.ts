/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AlignCenterOutline, AlignLeftOutline, AlignRightOutline } from "@makeplane/propel/icons";
import type { ComponentType, SVGProps } from "react";
// local imports
import { ECustomImageAttributeNames, ECustomImageStatus } from "./types";
import type { TCustomImageAlignment, Pixel, TCustomImageAttributes } from "./types";

export const DEFAULT_CUSTOM_IMAGE_ATTRIBUTES: TCustomImageAttributes = {
  [ECustomImageAttributeNames.SOURCE]: null,
  [ECustomImageAttributeNames.ID]: null,
  [ECustomImageAttributeNames.WIDTH]: "35%",
  [ECustomImageAttributeNames.HEIGHT]: "auto",
  [ECustomImageAttributeNames.ASPECT_RATIO]: null,
  [ECustomImageAttributeNames.ALIGNMENT]: "left",
  [ECustomImageAttributeNames.STATUS]: ECustomImageStatus.PENDING,
};

export const ensurePixelString = <TDefault>(
  value: Pixel | TDefault | number | undefined | null,
  defaultValue?: TDefault
) => {
  if (!value || value === defaultValue) {
    return defaultValue;
  }

  if (typeof value === "number") {
    return `${value}px` satisfies Pixel;
  }

  return value;
};

export const IMAGE_ALIGNMENT_OPTIONS: {
  label: string;
  value: TCustomImageAlignment;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
}[] = [
  {
    label: "Left",
    value: "left",
    icon: AlignLeftOutline,
  },
  {
    label: "Center",
    value: "center",
    icon: AlignCenterOutline,
  },
  {
    label: "Right",
    value: "right",
    icon: AlignRightOutline,
  },
];
export const getImageBlockId = (id: string) => `editor-image-block-${id}`;
