/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Handle } from "hast-util-to-mdast";

export const parseMarks: Record<string, Handle> = {
  // Markdown has no underline. Remove that wrapper, retaining all nested phrasing nodes.
  // Emphasis uses rehype-remark's normal handler, which also preserves nested marks.
  u: (state, node) => state.all(node),
};
