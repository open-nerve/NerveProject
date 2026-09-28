/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Components
export { TranslationProvider } from "./provider";

// Hooks
export { useTranslation } from "./hooks/use-translation";

// Utilities
export { setLanguage } from "./core/set-language";
export { initPromise } from "./core";

// Constants
export { FALLBACK_LANGUAGE, SUPPORTED_LANGUAGES, toSupportedLanguage } from "./constants/language";
