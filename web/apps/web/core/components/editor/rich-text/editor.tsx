/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef, lazy, Suspense } from "react";
import type { EditorRefApi } from "@nerve/editor";
import type { RichTextEditorWrapperProps } from "./types";

const LoadedEditor = lazy(() =>
  import("./editor-implementation").then((module) => ({ default: module.RichTextEditor }))
);

export const RichTextEditor = forwardRef<EditorRefApi, RichTextEditorWrapperProps>(function RichTextEditor(props, ref) {
  return (
    <Suspense fallback={null}>
      <LoadedEditor {...props} ref={ref} />
    </Suspense>
  );
});
