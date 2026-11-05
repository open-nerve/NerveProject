/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { EditorImages } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import { StoreContext } from "@/lib/store-context";
import { errorMessageKey } from "@/lib/error-messages";
import type { ContentView } from "@/store/asset/content-view";
import type { RootStore } from "@/store/root.store";
import { EditorAssets } from "./editor-assets";
export function useEditorAssets(view: ContentView | undefined, readOnly = false): EditorImages | undefined {
  const root = useContext(StoreContext);
  const { t } = useTranslation();
  const translate = useRef(t);
  translate.current = t;
  const [held, setHeld] = useState<{ root: RootStore; view: ContentView; owner: EditorAssets }>();
  useLayoutEffect(() => {
    if (!view?.current) return;
    const owner = new EditorAssets(root, view, (error) => translate.current(errorMessageKey(error)));
    setHeld({ root, view, owner });
    return owner.dispose;
  }, [root, view]);
  const images =
    view?.current && held?.root === root && held.view === view && !held.owner.images.signal.aborted
      ? held.owner.images
      : undefined;
  return useMemo(
    () => (images && readOnly ? { signal: images.signal, read: images.read, download: images.download } : images),
    [images, readOnly]
  );
}
