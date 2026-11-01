/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext } from "react";

import type { EditorLabels } from "@/types/config";

export const defaultEditorLabels: EditorLabels = {
  continueWriting: "Continue writing",
  resizeImage: "Resize image",
  tableRowOptions: "Table row options",
  tableColumnOptions: "Table column options",
  tableColors: {
    gray: "Gray",
    peach: "Peach",
    pink: "Pink",
    orange: "Orange",
    green: "Green",
    "light-blue": "Light blue",
    "dark-blue": "Dark blue",
    purple: "Purple",
    none: "No background color",
  },
};

export const EditorLabelsContext = createContext(defaultEditorLabels);
