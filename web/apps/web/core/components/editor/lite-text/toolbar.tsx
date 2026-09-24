/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useState, useCallback } from "react";

// editor
import type { EditorRefApi } from "@nerve/editor";
// i18n
import { useTranslation } from "@nerve/i18n";
// ui
import { Button } from "@nerve/propel/button";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// constants
import { cn } from "@nerve/utils";
import type { ToolbarMenuItem } from "@nerve/editor";
import { TOOLBAR_ITEMS } from "@nerve/editor";

type Props = {
  executeCommand: (item: ToolbarMenuItem) => void;
  handleSubmit: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
  isCommentEmpty: boolean;
  isSubmitting: boolean;
  showSubmitButton: boolean;
  editorRef: EditorRefApi | null;
  submitButtonText?: string;
};

export function IssueCommentToolbar(props: Props) {
  const { t } = useTranslation();
  const {
    executeCommand,
    handleSubmit,
    isCommentEmpty,
    isSubmitting,
    showSubmitButton,
    editorRef,
    submitButtonText = "common.comment",
  } = props;
  // State to manage active states of toolbar items
  const [activeStates, setActiveStates] = useState<Record<string, boolean>>({});

  // Function to update active states
  const updateActiveStates = useCallback(() => {
    if (!editorRef) return;
    const newActiveStates: Record<string, boolean> = {};
    Object.values(TOOLBAR_ITEMS)
      .flat()
      .forEach((item) => {
        // TODO: update this while toolbar homogenization
        // @ts-expect-error type mismatch here
        newActiveStates[item.renderKey] = editorRef.isMenuItemActive({
          itemKey: item.itemKey,
          ...item.extraProps,
        });
      });
    setActiveStates(newActiveStates);
  }, [editorRef]);

  // useEffect to call updateActiveStates when isActive prop changes
  useEffect(() => {
    if (!editorRef) return;
    const unsubscribe = editorRef.onStateChange(updateActiveStates);
    updateActiveStates();
    return () => unsubscribe();
  }, [editorRef, updateActiveStates]);

  const isEditorReadyToDiscard = editorRef?.isEditorReadyToDiscard();
  const isSubmitButtonDisabled = isCommentEmpty || !isEditorReadyToDiscard;

  return (
    <div className="flex h-9 w-full items-stretch gap-1.5 overflow-x-scroll bg-surface-2">
      <div className="flex w-full items-stretch justify-between gap-2 rounded-sm border-[0.5px] border-subtle p-1">
        <div className="flex items-stretch">
          {Object.keys(TOOLBAR_ITEMS).map((key, index) => (
            <div
              key={key}
              className={cn("flex items-stretch gap-0.5 border-r border-subtle px-2.5", {
                "pl-0": index === 0,
              })}
            >
              {TOOLBAR_ITEMS[key].map((item) => {
                const isItemActive = activeStates[item.renderKey];

                return (
                  <Tooltip key={item.renderKey} label={item.name} shortcut={item.shortcut?.join(" + ")}>
                    <button
                      type="button"
                      onClick={() => executeCommand(item)}
                      className={cn(
                        "grid aspect-square place-items-center rounded-xs p-0.5 text-placeholder hover:bg-layer-1",
                        {
                          "bg-layer-1 text-primary": isItemActive,
                        }
                      )}
                    >
                      <item.icon
                        className={cn("h-3.5 w-3.5", {
                          "text-primary": isItemActive,
                        })}
                        strokeWidth={2.5}
                      />
                    </button>
                  </Tooltip>
                );
              })}
            </div>
          ))}
        </div>
        {showSubmitButton && (
          <div className="sticky right-1">
            <Button
              type="submit"
              variant="primary"
              className="px-2.5 py-1.5 text-11"
              onClick={handleSubmit}
              disabled={isSubmitButtonDisabled}
              loading={isSubmitting}
            >
              {t(submitButtonText)}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
