/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Editor } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { afterEach, describe, expect, it } from "vitest";
// local imports
import { CoreEditorExtensions } from "@/extensions/extensions";
import { SideMenuExtension } from "@/extensions/side-menu";
import { SlashCommands } from "@/extensions/slash-commands";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import { CoreEditorProps } from "@/props";
import type { EditorRefApi, TFileHandler, TMentionHandler } from "@/types";

// The kept editors, driven the way a user drives them: typing, keys, the toolbar and the ref API the web
// app calls. Each editor is a real TipTap Editor built as useEditor builds the rich-text editor of a work
// item description: the same options, CoreEditorExtensions with the history on (the editor wrapper always
// passes enableHistory: true), then the side menu and the slash commands RichTextEditor adds. When the
// `editable` prop changes, useEditor destroys the editor and creates a new one, so "read-only, then
// editable again" is two editors here as well.

const fileHandler: TFileHandler = {
  assetsUploadStatus: {},
  cancel: () => {},
  checkIfAssetExists: async () => true,
  delete: async () => {},
  getAssetDownloadSrc: async (path) => path,
  getAssetSrc: async (path) => path,
  restore: async () => {},
  upload: async () => "asset-id",
  duplicate: async () => "asset-id",
  validation: { maxFileSize: 5 * 1024 * 1024 },
};

const ADA = { id: "user-ada", display_name: "Ada", url: "/profile/user-ada" };

const mentionHandler: TMentionHandler = {
  getMentionedEntityDetails: (id) => (id === ADA.id ? { display_name: ADA.display_name } : undefined),
  renderComponent: () => null,
};

const getEditorMetaData = () => ({
  file_assets: [{ id: "asset-1", name: "diagram.png", url: "https://files.example/asset-1" }],
  user_mentions: [ADA],
});

const openEditors: Editor[] = [];

afterEach(() => {
  for (const editor of openEditors.splice(0)) editor.destroy();
  document.body.replaceChildren();
});

const createEditor = async ({ editable = true, content = "" }: { editable?: boolean; content?: string } = {}) => {
  const element = document.createElement("div");
  document.body.append(element);
  const editor = new Editor({
    element,
    editable,
    parseOptions: { preserveWhitespace: true },
    editorProps: CoreEditorProps({ editorClassName: "" }),
    extensions: [
      ...CoreEditorExtensions({
        disabledExtensions: [],
        editable,
        enableHistory: true,
        fileHandler,
        getEditorMetaData,
        mentionHandler,
      }),
      SideMenuExtension({ dragDropEnabled: true }),
      SlashCommands({ disabledExtensions: [] }),
    ],
    content,
  });
  openEditors.push(editor);
  // the editor finishes setting up (initial ids, focus) in its "create" event, one task later
  await new Promise((resolve) => editor.on("create", resolve));
  return editor;
};

const refOf = (editor: Editor): EditorRefApi => getEditorRefHelpers({ editor, getEditorMetaData });

// Typed text, delivered as ProseMirror's DOM observer delivers it: each character goes to the
// handleTextInput props (the input rules) and is inserted at the selection when none of them takes it.
const typeText = (editor: Editor, text: string) => {
  for (const character of text) {
    const { view } = editor;
    const { from, to } = view.state.selection;
    const insert = () => view.state.tr.insertText(character, from, to);
    if (!view.someProp("handleTextInput", (handle) => handle(view, from, to, character, insert))) {
      view.dispatch(insert());
    }
  }
};

// A key press on the editor's DOM, where ProseMirror listens for keydown. Returns whether the editor
// handled it (a handled key press is default-prevented).
const press = (editor: Editor, key: string, modifiers: Omit<KeyboardEventInit, "key"> = {}) =>
  !editor.view.dom.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...modifiers }));

const parse = (html: string) => new DOMParser().parseFromString(html, "text/html").body;

// the block nodes of a document and the id each carries
const blockIds = (editor: Editor) => {
  const blocks: { type: string; id: unknown }[] = [];
  editor.state.doc.descendants((node: ProseMirrorNode) => {
    if (node.isBlock) blocks.push({ type: node.type.name, id: node.attrs.id });
  });
  return blocks;
};

const LEGACY_DESCRIPTION = "<p>Steps</p><ul><li><p>open the page</p></li></ul>";

describe("a rich-text editor built like the kept ones", () => {
  it("takes typed text and gives it back as HTML and as Markdown", async () => {
    const editor = await createEditor();

    typeText(editor, "# Release");
    press(editor, "Enter");
    typeText(editor, "Ship **today** and run `make`");

    const html = parse(editor.getHTML());
    expect(html.querySelector("h1")?.textContent).toBe("Release");
    expect(html.querySelector("p")?.textContent).toBe("Ship today and run make");
    expect(html.querySelector("p strong")?.textContent).toBe("today");
    expect(html.querySelector("p code")?.textContent).toBe("make");
    expect(refOf(editor).getMarkDown()).toBe("# Release\n\nShip **today** and run `make`\n");
  });

  it("undoes and redoes typing with the keyboard", async () => {
    const editor = await createEditor({ content: "<p>Draft</p>" });
    editor.commands.focus("end");

    typeText(editor, " ready");
    expect(editor.getText()).toBe("Draft ready");

    expect(press(editor, "z", { ctrlKey: true, keyCode: 90 })).toBe(true);
    expect(editor.getText()).toBe("Draft");

    // with Shift held the browser reports the key as "Z"
    expect(press(editor, "Z", { ctrlKey: true, shiftKey: true, keyCode: 90 })).toBe(true);
    expect(editor.getText()).toBe("Draft ready");

    refOf(editor).undo();
    expect(editor.getText()).toBe("Draft");
    refOf(editor).redo();
    expect(editor.getText()).toBe("Draft ready");
  });

  it("shows a description read-only, and edits it once it is editable again", async () => {
    const readOnly = await createEditor({ editable: false, content: LEGACY_DESCRIPTION });
    readOnly.commands.focus("end");
    const shown = readOnly.getJSON();

    expect(readOnly.view.dom.getAttribute("contenteditable")).toBe("false");
    expect(press(readOnly, "Enter")).toBe(false);
    expect(press(readOnly, "Backspace")).toBe(false);
    expect(readOnly.getJSON()).toEqual(shown);
    // read-only leaves the saved description as it is: no ids are added to it either
    expect(blockIds(readOnly).every(({ id }) => id === null)).toBe(true);
    const saved = readOnly.getHTML();
    // the editable prop flips: useEditor destroys this editor and creates a new one
    readOnly.destroy();

    const editable = await createEditor({ content: saved });
    editable.commands.focus("end");
    expect(editable.view.dom.getAttribute("contenteditable")).toBe("true");
    expect(press(editable, "Enter")).toBe(true);
    typeText(editable, "check the result");
    expect(parse(editable.getHTML()).querySelectorAll("li")).toHaveLength(2);
    expect(editable.getText()).toContain("check the result");
  });

  it("inserts a user mention that is saved, read back and copied as Markdown", async () => {
    const editor = await createEditor({ content: "<p>Ask </p>" });
    editor.commands.focus("end");

    // what the mention dropdown's command inserts for the chosen user
    editor.commands.insertContent([
      {
        type: "mention",
        attrs: { id: "mention-1", entity_identifier: ADA.id, entity_name: "user_mention" },
      },
      { type: "text", text: " " },
    ]);

    const mention = parse(editor.getHTML()).querySelector("mention-component");
    expect(mention?.getAttribute("entity_identifier")).toBe(ADA.id);
    expect(mention?.getAttribute("entity_name")).toBe("user_mention");
    expect(editor.getText()).toBe("Ask @Ada ");
    expect(refOf(editor).getMarkDown()).toContain(`[@Ada](${ADA.url})`);

    const reopened = await createEditor({ content: editor.getHTML() });
    const mentions: ProseMirrorNode[] = [];
    reopened.state.doc.descendants((node: ProseMirrorNode) => {
      if (node.type.name === "mention") mentions.push(node);
    });
    expect(mentions.map((node) => node.attrs.entity_identifier)).toEqual([ADA.id]);
  });

  it("inserts an image from the toolbar, and keeps an uploaded image through a save", async () => {
    const editor = await createEditor({ content: "<p>Screenshot:</p>" });
    editor.commands.focus("end");

    refOf(editor).executeMenuItemCommand({ itemKey: "image", savedSelection: null });

    const images: ProseMirrorNode[] = [];
    editor.state.doc.descendants((node: ProseMirrorNode) => {
      if (node.type.name === "imageComponent") images.push(node);
    });
    expect(images).toHaveLength(1);
    expect(images[0].attrs.status).toBe("pending");
    expect(images[0].attrs.id).toEqual(expect.any(String));
    expect(parse(editor.getHTML()).querySelector("image-component")?.getAttribute("status")).toBe("pending");

    const uploaded =
      '<image-component src="asset-1" width="35%" alignment="center" status="uploaded"></image-component>';
    const reopened = await createEditor({ content: `<p>Screenshot:</p>${uploaded}` });
    const image = parse(reopened.getHTML()).querySelector("image-component");
    expect(image?.getAttribute("src")).toBe("asset-1");
    expect(image?.getAttribute("alignment")).toBe("center");
    expect(refOf(reopened).getMarkDown()).toContain("![diagram.png](https://files.example/asset-1)");
  });

  it("gives every block node an id, keeps it through a save and gives new blocks new ones", async () => {
    const editor = await createEditor({ content: LEGACY_DESCRIPTION });

    const initial = blockIds(editor);
    expect(initial.map(({ type }) => type)).toEqual(["paragraph", "bulletList", "listItem", "paragraph"]);
    expect(initial.every(({ id }) => typeof id === "string")).toBe(true);
    expect(new Set(initial.map(({ id }) => id)).size).toBe(initial.length);

    editor.commands.focus("end");
    press(editor, "Enter");
    press(editor, "Enter");
    typeText(editor, "Expected");
    const after = blockIds(editor);
    const added = after.filter(({ id }) => !initial.some((block) => block.id === id));
    expect(added.map(({ type }) => type)).toEqual(["paragraph"]);
    expect(added[0].id).toEqual(expect.any(String));

    const saved = editor.getHTML();
    expect(parse(saved).querySelectorAll("[data-id]")).toHaveLength(after.length);
    const reopened = await createEditor({ content: saved });
    expect(blockIds(reopened)).toEqual(after);
  });

  it("creates editors one after another in one process: editable, read-only, editable", async () => {
    const first = await createEditor({ content: LEGACY_DESCRIPTION });
    expect(blockIds(first).every(({ id }) => typeof id === "string")).toBe(true);

    const readOnly = await createEditor({ editable: false, content: LEGACY_DESCRIPTION });
    expect(blockIds(readOnly).every(({ id }) => id === null)).toBe(true);

    const third = await createEditor({ content: LEGACY_DESCRIPTION });
    expect(blockIds(third).every(({ id }) => typeof id === "string")).toBe(true);
    const initialBlocks = blockIds(third).length;
    third.commands.focus("end");
    press(third, "Enter");
    press(third, "Enter");
    typeText(third, "Notes");
    expect(blockIds(third).length).toBeGreaterThan(initialBlocks);
    expect(blockIds(third).every(({ id }) => typeof id === "string")).toBe(true);

    // the editors share nothing: typing and undo in one leave the other alone
    first.commands.focus("end");
    typeText(first, " now");
    expect(first.getText()).toContain("open the page now");
    expect(third.getText()).not.toContain("now");
    third.commands.focus("end");
    press(third, "z", { ctrlKey: true, keyCode: 90 });
    expect(third.getText()).not.toContain("Notes");
    expect(first.getText()).toContain("open the page now");
  });
});
