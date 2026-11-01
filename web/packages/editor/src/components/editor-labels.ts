/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext } from "react";

import type { EditorLabels } from "@/types/config";

export const defaultEditorLabels: EditorLabels = {
  continueWriting: "Continue writing",
  resizeImage: "Resize image",
};

export const EditorLabelsContext = createContext(defaultEditorLabels);
