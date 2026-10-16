/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { TwitterPicker } from "react-color";
import type { TwitterPickerProps } from "react-color";

// react-color draws its hex field with an inline outline: none, which hides where the keyboard is. "revert" gives the
// field the browser's own outline back, which shows as the keyboard reaches it.
const STYLES = { default: { input: { outline: "revert" } } };

/** react-color's TwitterPicker, whose hex field shows the keyboard's focus. */
export function ColorPicker(props: TwitterPickerProps) {
  return <TwitterPicker {...props} styles={STYLES} />;
}
