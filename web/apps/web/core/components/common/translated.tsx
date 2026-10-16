/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { Children } from "react";
import type { ReactNode } from "react";
import { useTranslation } from "@nerve/i18n";

// what stands in the translated text for a placeholder's node: no translation holds it
const MARK = "\u0000";

type Props = {
  /** The key of a sentence with placeholders, such as "Delete {name}?". */
  i18nKey: string;
  /** The node of each placeholder, such as the name set in bold. */
  values: Record<string, ReactNode>;
};

/**
 * A sentence in the page's language whose placeholders are nodes, not text: one key for the whole sentence, so that
 * each language orders it as it does, and the thing it names set apart within it.
 */
export function Translated(props: Props) {
  const { i18nKey, values } = props;
  const { t } = useTranslation();
  const names = Object.keys(values);
  const text = t(i18nKey, Object.fromEntries(names.map((name) => [name, `${MARK}${name}${MARK}`])));
  // the marks split the text into its own parts and, at odd places, the placeholders' names; Children keys the parts
  return Children.toArray(text.split(MARK).map((part, index) => (index % 2 === 1 ? values[part] : part)));
}
