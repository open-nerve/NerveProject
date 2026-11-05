/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useImperativeHandle, useRef, useState, type KeyboardEvent, type Ref } from "react";
/** Shared text edge for comment consumer tests; real forms/stores/commands remain mounted. */
export function commentEditor(ready: () => boolean = () => true) {
  return function CommentEditor({
    id,
    initialValue,
    onChange,
    isSubmitting,
    onEnterKeyPress,
    editorProps,
    editable = true,
    ref,
  }: {
    id: string;
    initialValue: string;
    onChange?: (json: unknown, html: string) => void;
    isSubmitting?: boolean;
    onEnterKeyPress?: (event?: KeyboardEvent) => void;
    editorProps?: { attributes: Record<string, string> };
    editable?: boolean;
    ref?: Ref<{
      clearEditor(): void;
      setEditorValue(html: string): void;
      isEditorReadyToDiscard(): boolean;
      focus(position?: unknown): void;
    }>;
  }) {
    const field = useRef<HTMLTextAreaElement>(null);
    const [text, setText] = useState(initialValue);
    useImperativeHandle(ref, () => ({
      clearEditor: () => setText(""),
      setEditorValue: setText,
      isEditorReadyToDiscard: ready,
      focus: () => field.current?.focus(),
    }));
    return (
      <textarea
        ref={field}
        {...editorProps?.attributes}
        id={id}
        data-editor
        value={text}
        disabled={isSubmitting}
        readOnly={!editable}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.shiftKey && !event.ctrlKey && !event.metaKey) onEnterKeyPress?.(event);
        }}
        onChange={(event) => {
          setText(event.target.value);
          onChange?.(undefined, event.target.value);
        }}
      />
    );
  };
}
