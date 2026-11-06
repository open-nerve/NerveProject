/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext } from "react";

import type { EditorLabels } from "@/types/config";

export const defaultEditorLabels: EditorLabels = {
  image: {
    select: "Choose images",
    cancel: "Cancel",
    retry: "Retry",
    unavailable: "Image unavailable",
    loading: "Loading image",
    copyUnknown:
      "Copying may have created an unfinished file. Check the draft files before publishing; retrying may create another copy.",
    cancelled: "This image's transfer was cancelled. Choose the image again.",
    invalid: "Choose only JPEG, PNG, WebP or GIF images.",
    size: "The image is empty or exceeds the current file size limit.",
    name: "The image filename is invalid.",
    tooMany: "Choose up to 10 images at once.",
    queued: "Waiting to upload",
    preparing: "Preparing image",
    uploading: "Uploading image",
    confirming: "Confirming image",
    download: "Download image",
    downloading: "Downloading image",
    downloadFailed: "The image could not be downloaded. Try again.",
    fullScreen: "View image in full screen",
    close: "Close image viewer",
    zoomIn: "Zoom in",
    zoomOut: "Zoom out",
    align: "Align image",
    left: "Align left",
    center: "Align center",
    right: "Align right",
  },
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
