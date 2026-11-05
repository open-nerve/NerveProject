/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IRichTextEditorProps } from "@nerve/editor";
import type { MakeOptional } from "@nerve/types";

export type RichTextEditorWrapperProps = MakeOptional<
  Omit<IRichTextEditorProps, "mentionHandler">,
  "disabledExtensions" | "editable" | "getEditorMetaData"
> & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  issueSequenceId?: number;
};
