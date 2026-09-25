/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { TLogoProps } from "@nerve/types";
// types
import type { TCalloutBlockAttributes, TCalloutBlockEmojiAttributes, TCalloutBlockIconAttributes } from "./types";
import { ECalloutAttributeNames } from "./types";

export const DEFAULT_CALLOUT_BLOCK_ATTRIBUTES: TCalloutBlockAttributes = {
  [ECalloutAttributeNames.ID]: null,
  [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
  [ECalloutAttributeNames.ICON_COLOR]: undefined,
  [ECalloutAttributeNames.ICON_NAME]: undefined,
  [ECalloutAttributeNames.EMOJI_UNICODE]: "128161",
  [ECalloutAttributeNames.EMOJI_URL]: "https://cdn.jsdelivr.net/npm/emoji-datasource-apple/img/apple/64/1f4a1.png",
  [ECalloutAttributeNames.BACKGROUND]: undefined,
  [ECalloutAttributeNames.BLOCK_TYPE]: "callout-component",
};

type TStoredLogoValue = Pick<TCalloutBlockAttributes, ECalloutAttributeNames.LOGO_IN_USE> &
  (TCalloutBlockEmojiAttributes | TCalloutBlockIconAttributes);

const isRecord = (value: unknown): value is Record<string, unknown> => typeof value === "object" && value !== null;

const isNonEmptyString = (value: unknown): value is string => typeof value === "string" && value !== "";

const isOptionalString = (value: unknown): value is string | undefined =>
  value === undefined || typeof value === "string";

// the callout attributes of a stored logo, or undefined when the stored text is not a logo the selector writes:
// local storage holds whatever was put there, so every field the callout reads is checked, not assumed
const storedLogoAttributes = (storedData: string): TStoredLogoValue | undefined => {
  let parsedData: unknown;
  try {
    parsedData = JSON.parse(storedData);
  } catch {
    return undefined;
  }
  if (!isRecord(parsedData)) return undefined;
  const { in_use, emoji, icon } = parsedData;
  if (in_use === "emoji" && isRecord(emoji) && isNonEmptyString(emoji.value) && isOptionalString(emoji.url)) {
    return {
      [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
      [ECalloutAttributeNames.EMOJI_UNICODE]: emoji.value,
      [ECalloutAttributeNames.EMOJI_URL]:
        emoji.url || DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_URL],
    };
  }
  if (in_use === "icon" && isRecord(icon) && isNonEmptyString(icon.name) && isOptionalString(icon.color)) {
    return {
      [ECalloutAttributeNames.LOGO_IN_USE]: "icon",
      [ECalloutAttributeNames.ICON_NAME]: icon.name,
      [ECalloutAttributeNames.ICON_COLOR]:
        icon.color || DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.ICON_COLOR],
    };
  }
  return undefined;
};

// function to get the stored logo from local storage; a stored value that is not a logo is dropped
export const getStoredLogo = (): TStoredLogoValue => {
  const fallBackValues: TStoredLogoValue = {
    [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
    [ECalloutAttributeNames.EMOJI_UNICODE]: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_UNICODE],
    [ECalloutAttributeNames.EMOJI_URL]: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_URL],
  };

  if (typeof window !== "undefined") {
    const storedData = localStorage.getItem("editor-calloutComponent-logo");
    if (storedData) {
      const storedLogo = storedLogoAttributes(storedData);
      if (storedLogo) return storedLogo;
      console.error(`Invalid stored callout logo, stored value- ${storedData}`);
      localStorage.removeItem("editor-calloutComponent-logo");
    }
  }
  // fallback values
  return fallBackValues;
};
// function to update the stored logo on local storage
export const updateStoredLogo = (value: TLogoProps): void => {
  if (typeof window === "undefined") return;
  localStorage.setItem("editor-calloutComponent-logo", JSON.stringify(value));
};
// function to get the stored background color from local storage
export const getStoredBackgroundColor = (): string | null => {
  if (typeof window !== "undefined") {
    return localStorage.getItem("editor-calloutComponent-background") ?? "";
  }
  return null;
};
// function to update the stored background color on local storage
export const updateStoredBackgroundColor = (value: string | null): void => {
  if (typeof window === "undefined") return;
  if (value === null) {
    localStorage.removeItem("editor-calloutComponent-background");
    return;
  } else {
    localStorage.setItem("editor-calloutComponent-background", value);
  }
};
