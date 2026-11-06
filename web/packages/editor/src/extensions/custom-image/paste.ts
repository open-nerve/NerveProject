/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Fragment, Slice, type Node } from "@tiptap/pm/model";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { v4 as uuid } from "uuid";
import type { Editor } from "@tiptap/core";
import type { CustomImageExtensionStorage } from "./types";
import { isImageID } from "./input";
/** Clipboard IDs are untrusted references, even when the sender claims an uploaded/same-document node. */
export function imagePaste(editor: Editor, storage: CustomImageExtensionStorage) {
  let moving = false;
  return new Plugin({
    key: new PluginKey("image-paste"),
    appendTransaction(transactions, _old, state) {
      if (!transactions.some((tr) => tr.docChanged)) return null;
      const tr = state.tr;
      state.doc.descendants((node, pos) => {
        const id = node.attrs.id;
        const completed = storage.completed.get(id);
        if (node.type.name === "imageComponent" && !node.attrs.src && completed)
          tr.setNodeMarkup(pos, undefined, { ...node.attrs, src: completed, status: "uploaded" });
      });
      return tr.docChanged ? tr.setMeta("addToHistory", false) : null;
    },
    props: {
      handleDOMEvents: {
        drop(view, event) {
          // Only an actual in-memory drag can preserve ownership. Clipboard attributes never enter this branch.
          const copies =
            view.someProp("dragCopies", (test) => test(event)) ??
            (/Mac|iP(hone|ad|od)/.test(navigator.platform) ? event.altKey : event.ctrlKey);
          moving = !copies && !!view.dragging;
          queueMicrotask(() => {
            moving = false;
          });
          return false;
        },
        paste() {
          moving = false;
          return false;
        },
      },
      transformPasted(slice) {
        if (moving) {
          moving = false;
          return slice;
        }
        const map = (fragment: Fragment): Fragment => {
          const nodes: Node[] = [];
          fragment.forEach((node) => {
            if (node.type.name === "image") return;
            if (node.type.name === "imageComponent") {
              if (
                !editor.isEditable ||
                !storage.images?.write ||
                storage.images.signal.aborted ||
                !isImageID(node.attrs.src)
              )
                return;
              const id = uuid();
              storage.fileMap.set(id, { sourceID: node.attrs.src, autoStart: true });
              nodes.push(node.type.create({ ...node.attrs, id, src: null, status: "pending" }));
            } else nodes.push(node.isText ? node : node.copy(map(node.content)));
          });
          return Fragment.from(nodes);
        };
        return new Slice(map(slice.content), slice.openStart, slice.openEnd);
      },
    },
    view() {
      return {
        destroy() {
          storage.transfers.destroy();
          for (const id of storage.fileMap.keys()) storage.images?.write?.cancel(id);
          storage.fileMap.clear();
          storage.completed.clear();
        },
      };
    },
  });
}
