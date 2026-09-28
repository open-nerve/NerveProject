/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { initPromise, i18nInstance } from "./instance";
import { LANGUAGE_STORAGE_KEY } from "../constants/language";
import type { TLanguage } from "../types";

/**
 * Sets the page's language: i18next's, then the stored language and <html lang>. i18next applies only the language
 * asked for last, and a call can finish after a later one: a call whose language i18next did not apply writes
 * nothing, and the call whose language it applied writes it.
 */
export async function setLanguage(lng: TLanguage): Promise<void> {
  await initPromise;
  await i18nInstance.changeLanguage(lng);
  if (i18nInstance.language !== lng) return;
  if (typeof window !== "undefined") {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, lng);
    document.documentElement.lang = lng;
  }
}
