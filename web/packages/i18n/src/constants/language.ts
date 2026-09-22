/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TLanguage, ILanguageOption } from "../types";

export const FALLBACK_LANGUAGE: TLanguage = "en";

export const SUPPORTED_LANGUAGES: ILanguageOption[] = [
  { label: "English", value: "en" },
  { label: "简体中文", value: "zh-CN" },
];

export const LANGUAGE_STORAGE_KEY = "userLanguage";

/**
 * Returns the language if it is supported, otherwise the fallback language. A language kept in
 * local storage or in the user's profile can be one that is no longer shipped, such as "fr".
 */
export function toSupportedLanguage(language: string | null | undefined): TLanguage {
  return SUPPORTED_LANGUAGES.find((option) => option.value === language)?.value ?? FALLBACK_LANGUAGE;
}
