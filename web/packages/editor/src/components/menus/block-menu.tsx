/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import {
  useFloating,
  autoUpdate,
  offset,
  flip,
  shift,
  useDismiss,
  useInteractions,
  FloatingPortal,
  FloatingFocusManager,
} from "@floating-ui/react";
import type { Editor } from "@tiptap/react";
import type { LucideIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { CopyOutline, DeleteOutline } from "@makeplane/propel/icons";
import type { ISvgIcons } from "@nerve/propel/icons";
import { cn, isComposing } from "@nerve/utils";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";

type Props = {
  editor: Editor;
};
type BlockMenuOption = {
  icon: LucideIcon | React.FC<ISvgIcons>;
  key: string;
  label: string;
  onClick: (e: React.MouseEvent) => void;
  isDisabled?: boolean;
};

export function BlockMenu(props: Props) {
  const { editor } = props;
  const [openedFor, setOpenedFor] = useState<Editor | null>(null);
  const isOpen = openedFor === editor && editor.isEditable && !editor.isDestroyed;
  const setIsOpen = useCallback((open: boolean) => setOpenedFor(open ? editor : null), [editor]);
  const [isAnimatedIn, setIsAnimatedIn] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const handleRef = useRef<HTMLElement | null>(null);
  const firstActionRef = useRef<HTMLButtonElement | null>(null);

  // The actual handle is both the positioning anchor and the keyboard return target.
  const { refs, floatingStyles, context } = useFloating({
    open: isOpen,
    onOpenChange: setIsOpen,
    middleware: [offset({ crossAxis: -10 }), flip(), shift()],
    whileElementsMounted: autoUpdate,
    placement: "left-start",
  });

  const dismiss = useDismiss(context, { escapeKey: false });
  const { getFloatingProps } = useInteractions([dismiss]);

  const openBlockMenu = useCallback(() => {
    if (!isOpen) {
      setIsOpen(true);
    }
  }, [isOpen, setIsOpen]);

  const closeBlockMenu = useCallback(() => {
    if (isOpen) {
      setIsOpen(false);
    }
  }, [isOpen, setIsOpen]);

  useEffect(() => {
    if (!isOpen) return;
    editor.commands.addActiveDropbarExtension(CORE_EXTENSIONS.SIDE_MENU);
    return () => {
      if (!editor.isDestroyed) editor.commands.removeActiveDropbarExtension(CORE_EXTENSIONS.SIDE_MENU);
    };
  }, [editor, isOpen]);

  // Handle click on drag handle
  const handleClickDragHandle = useCallback(
    (event: MouseEvent) => {
      const target = event.target;
      if (!(target instanceof Element) || editor.isDestroyed) return;
      const dragHandle = target.closest("#drag-handle");

      // The side-menu plugin mounts its handle beside this editor's view. Several editors
      // can be mounted together, so a document event is not by itself proof of ownership.
      if (
        dragHandle instanceof HTMLElement &&
        editor.isEditable &&
        dragHandle.closest("#editor-side-menu")?.parentElement === editor.view.dom.parentElement
      ) {
        event.preventDefault();

        handleRef.current = dragHandle;
        refs.setReference(dragHandle);

        // Show the menu
        openBlockMenu();
        return;
      }

      // If clicking outside and not on a menu item, hide the menu
      if (menuRef.current && !menuRef.current.contains(target)) {
        closeBlockMenu();
      }
    },
    [editor, refs, openBlockMenu, closeBlockMenu]
  );

  // Set up event listeners
  useEffect(() => {
    const handleScroll = () => {
      closeBlockMenu();
    };

    document.addEventListener("click", handleClickDragHandle);
    document.addEventListener("contextmenu", handleClickDragHandle);
    document.addEventListener("scroll", handleScroll, true);

    return () => {
      document.removeEventListener("click", handleClickDragHandle);
      document.removeEventListener("contextmenu", handleClickDragHandle);
      document.removeEventListener("scroll", handleScroll, true);
    };
  }, [editor.commands, handleClickDragHandle, closeBlockMenu]);

  // Animation effect
  useEffect(() => {
    if (isOpen) {
      setIsAnimatedIn(false);
      let frame: number | undefined;
      // Add a small delay before starting the animation
      const timeout = setTimeout(() => {
        frame = requestAnimationFrame(() => {
          setIsAnimatedIn(true);
        });
      }, 50);

      return () => {
        clearTimeout(timeout);
        if (frame !== undefined) cancelAnimationFrame(frame);
      };
    } else {
      setIsAnimatedIn(false);
    }
  }, [isOpen]);

  const MENU_ITEMS: BlockMenuOption[] = [
    {
      icon: DeleteOutline,
      key: "delete",
      label: "Delete",
      onClick: (_e) => {
        // Execute the delete action
        editor.chain().deleteSelection().focus().run();
      },
    },
    {
      icon: CopyOutline,
      key: "duplicate",
      label: "Duplicate",
      isDisabled:
        editor.state.selection.content().content.firstChild?.type.name === CORE_EXTENSIONS.IMAGE ||
        editor.isActive(CORE_EXTENSIONS.CUSTOM_IMAGE),
      onClick: (_e) => {
        try {
          const { state } = editor;
          const { selection } = state;
          const firstChild = selection.content().content.firstChild;
          const docSize = state.doc.content.size;

          if (!firstChild) {
            throw new Error("No content selected or content is not duplicable.");
          }

          // Directly use selection.to as the insertion position
          const insertPos = selection.to;

          // Ensure the insertion position is within the document's bounds
          if (insertPos < 0 || insertPos > docSize) {
            throw new Error("The insertion position is invalid or outside the document.");
          }

          const contentToInsert = firstChild.toJSON();

          // Insert the content at the calculated position
          editor
            .chain()
            .insertContentAt(insertPos, contentToInsert, {
              updateSelection: true,
            })
            // insertContentAt already resolves a valid selection in the updated document.
            // Clamping a text position against the old size can land between block nodes.
            .focus(null, { scrollIntoView: false })
            .run();
        } catch (error) {
          if (error instanceof Error) {
            console.error(error.message);
          }
        }
      },
    },
  ];

  if (!isOpen) {
    return null;
  }

  return (
    <FloatingPortal>
      <FloatingFocusManager context={context} initialFocus={firstActionRef} returnFocus={handleRef}>
        <div
          role="group"
          aria-label="Block actions"
          ref={(node) => {
            refs.setFloating(node);
            menuRef.current = node;
          }}
          style={{
            ...floatingStyles,
            animationFillMode: "forwards",
            transitionTimingFunction: "cubic-bezier(0.16, 1, 0.3, 1)", // Expo ease out
            zIndex: 100,
          }}
          className={cn(
            "max-h-60 min-w-[7rem] overflow-y-scroll rounded-lg border border-subtle bg-surface-1 p-1.5 shadow-raised-200",
            "origin-top-right transform transition-all duration-300",
            isAnimatedIn ? "scale-100 opacity-100" : "scale-75 opacity-0"
          )}
          {...getFloatingProps({
            onKeyDown: (event: React.KeyboardEvent) => {
              if (event.key !== "Escape") return;
              // An inner menu owns Escape, including an IME's Escape that must not dismiss either layer.
              event.stopPropagation();
              if (isComposing(event.nativeEvent)) return;
              event.preventDefault();
              closeBlockMenu();
            },
          })}
        >
          {MENU_ITEMS.map((item) => {
            if (item.isDisabled) return null;

            return (
              <button
                key={item.key}
                ref={item.key === "delete" ? firstActionRef : undefined}
                type="button"
                className="focus-visible:outline-accent-primary flex w-full items-center gap-1.5 truncate rounded-sm px-1 py-1.5 text-11 text-secondary hover:bg-layer-1 focus-visible:outline-2"
                onClick={(e) => {
                  item.onClick(e);
                  e.preventDefault();
                  e.stopPropagation();
                  closeBlockMenu();
                }}
                disabled={item.isDisabled}
              >
                <item.icon className="h-3 w-3" />
                {item.label}
              </button>
            );
          })}
        </div>
      </FloatingFocusManager>
    </FloatingPortal>
  );
}
