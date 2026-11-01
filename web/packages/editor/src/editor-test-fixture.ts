/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Editor } from "@tiptap/core";
import type { EditorProps } from "@tiptap/pm/view";
import { CoreEditorExtensions } from "@/extensions/extensions";
import { SideMenuExtension } from "@/extensions/side-menu";
import { SlashCommands } from "@/extensions/slash-commands";
import { CoreEditorProps } from "@/props";
import type { TExtensions, TFileHandler, TMentionHandler } from "@/types";

export const fileHandler: TFileHandler = {
  assetsUploadStatus: {},
  delete: async () => {},
  getAssetDownloadSrc: async (path) => path,
  getAssetSrc: async (path) => path,
  restore: async () => {},
  upload: async () => "asset-id",
  duplicate: async () => "asset-id",
  validation: { maxFileSize: 5 * 1024 * 1024 },
};

export const ADA = { id: "user-ada", display_name: "Ada", url: "/profile/user-ada" };

const mentionHandler: TMentionHandler = {
  getMentionedEntityDetails: (id) => (id === ADA.id ? { display_name: ADA.display_name } : undefined),
  renderComponent: () => null,
};

export const getEditorMetaData = () => ({
  file_assets: [{ id: "asset-1", name: "diagram.png", url: "https://files.example/asset-1" }],
  user_mentions: [ADA],
});

const openEditors: Editor[] = [];

export function destroyEditors() {
  for (const editor of openEditors.splice(0)) editor.destroy();
}

export const createEditor = async ({
  editable = true,
  content = "",
  disabledExtensions = [],
  attributes,
}: {
  editable?: boolean;
  content?: string;
  disabledExtensions?: TExtensions[];
  attributes?: EditorProps["attributes"];
} = {}) => {
  const element = document.createElement("div");
  document.body.append(element);
  const editor = new Editor({
    element,
    editable,
    parseOptions: { preserveWhitespace: true },
    editorProps: CoreEditorProps({ editorClassName: "consumer-editor", attributes }),
    extensions: [
      ...CoreEditorExtensions({
        disabledExtensions,
        editable,
        enableHistory: true,
        fileHandler,
        getEditorMetaData,
        mentionHandler,
      }),
      SideMenuExtension({ dragDropEnabled: true }),
      SlashCommands({ disabledExtensions }),
    ],
    content,
  });
  openEditors.push(editor);
  // the editor finishes setting up (initial ids, focus) in its "create" event, one task later
  await new Promise((resolve) => editor.on("create", resolve));
  return editor;
};
