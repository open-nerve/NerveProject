/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useImperativeHandle, useLayoutEffect, useMemo, useRef, useState } from "react";
import { AutosaveError, PreservedInput, useConfirmedAutosave } from "../../use-confirmed-autosave";
import { useDescriptionRetention } from "./retention";
import { contentImageIDs, verifyEditorImages } from "@/hooks/editor/verify-images";
import { observer } from "mobx-react";
// nerve imports
import type { EditorImages, EditorRefApi, TExtensions } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
import { getDescriptionPlaceholderI18n } from "@nerve/utils";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// local imports
type TFormData = {
  id: string;
  description_html: string;
};

export type DescriptionInputHandle = { restore: (confirm: () => Promise<string>) => Promise<void> };
type Props = {
  autosaveRef?: React.Ref<DescriptionInputHandle>;
  images?: EditorImages;
  prepareSubmit?: () => Props["onSubmit"];
  /**
   * @description Container class name, this will be used to add custom styles to the editor container
   */
  containerClassName?: string;
  /**
   * @description Disabled, this will be used to disable the editor
   */
  disabled?: boolean;
  /**
   * @description Disabled extensions, this will be used to disable the extensions in the editor
   */
  disabledExtensions?: TExtensions[];
  /**
   * @description Editor ref, this will be used to imperatively attach editor related helper functions
   */
  editorRef?: React.RefObject<EditorRefApi | null>;
  /**
   * @description Entity ID, this will be used for file uploads and as the unique identifier for the entity
   */
  entityId: string;
  /**
   * @description Initial value, pass the actual description to initialize the editor
   */
  initialValue: string | undefined;
  /**
   * @description Submit handler, the actual function which will be called when the form is submitted
   */
  onSubmit: (
    value: {
      description_html: string;
    },
    isMigrationUpdate?: boolean
  ) => Promise<void>;
  /**
   * @description Placeholder, if not provided, the placeholder will be the default placeholder
   */
  placeholder?: string | ((isFocused: boolean, value: string) => string);
  /**
   * @description projectId, if not provided, the entity will be considered as a workspace entity
   */
  projectId?: string;
  /**
   * @description Set is submitting, use it to set the loading state of the form
   */
  setIsSubmitting: (initialValue: TNameDescriptionLoader) => void;
  /**
   * @description SWR description, use it only if you want to sync changes in realtime(pseudo realtime)
   */
  swrDescription?: string | null | undefined;
  /**
   * @description Workspace slug, this will be used to get the workspace details
   */
  workspaceSlug: string;
  /**
   * @description Issue sequence id, this will be used to get the issue sequence id
   */
  issueSequenceId?: number;
};

/**
 * @description DescriptionInput component for rich text editor with autosave functionality using debounce
 * The component also makes an API call to save the description on unmount
 */
export function DescriptionInput(props: Props) {
  return (
    <DescriptionInputContent key={JSON.stringify([props.workspaceSlug, props.projectId, props.entityId])} {...props} />
  );
}
const sameDescription = (left: TFormData, right: TFormData) =>
  left.id === right.id && left.description_html === right.description_html;
const DescriptionInputContent = observer(function DescriptionInputContent(props: Props) {
  const {
    autosaveRef,
    prepareSubmit,
    containerClassName,
    disabled,
    disabledExtensions,
    editorRef: givenEditorRef,
    images,
    entityId,
    initialValue,
    issueSequenceId,
    onSubmit,
    placeholder,
    projectId,
    setIsSubmitting,
    swrDescription,
    workspaceSlug,
  } = props;
  const ownedEditorRef = useRef<EditorRefApi>(null);
  const editorRef = givenEditorRef ?? ownedEditorRef;
  const retention = useDescriptionRetention(entityId);
  const { getWorkspaceBySlug } = useWorkspace();
  const workspaceDetails = getWorkspaceBySlug(workspaceSlug);
  const { t } = useTranslation();
  const initial = useMemo(
    () => ({ id: entityId, description_html: initialValue ?? "", isMigrationUpdate: false }),
    [entityId, initialValue]
  );
  const {
    value: localDescription,
    change,
    failure,
    retry,
    replaceWith,
    replacing,
    conflict,
    needsResume,
    resume,
    discard,
  } = useConfirmedAutosave({
    initial,
    same: sameDescription,
    save: (input) => onSubmit({ description_html: input.description_html }, input.isMigrationUpdate),
    prepareSave: prepareSubmit
      ? () => {
          const submit = prepareSubmit();
          return (input) => submit({ description_html: input.description_html }, input.isMigrationUpdate);
        }
      : undefined,
    status: setIsSubmitting,
    disabled,
    retention,
    canSave: (input) => !contentImageIDs(input.description_html).incomplete,
    // Retained image identities are resent only after the current view verifies them (recover below).
    resumesOnEdit: (input) => contentImageIDs(input.description_html).ids.length === 0,
  });
  const [verifying, setVerifying] = useState(false);
  const [imageFailure, setImageFailure] = useState(false);
  // What the last committed render showed; a verification that outlives it must not resume newer input.
  const current = useRef({ active: false, html: localDescription.description_html, images });
  useLayoutEffect(() => {
    current.current.html = localDescription.description_html;
    current.current.images = images;
  }, [localDescription.description_html, images]);
  useLayoutEffect(() => {
    const mounted = current.current;
    mounted.active = true;
    return () => {
      mounted.active = false;
    };
  }, []);
  const unfinished = contentImageIDs(localDescription.description_html).incomplete;
  const recover = async () => {
    // An image still transferring is reported as unfinished below and verified once it finishes.
    if (verifying || unfinished) return;
    const html = localDescription.description_html;
    const live = () => current.current.active && current.current.images === images && current.current.html === html;
    setVerifying(true);
    setImageFailure(false);
    try {
      await verifyEditorImages(images, html);
      if (live()) resume();
    } catch {
      if (live()) setImageFailure(true);
    } finally {
      if (current.current.active) setVerifying(false);
    }
  };

  useImperativeHandle(
    autosaveRef,
    () => ({
      restore: async (confirm) => {
        const result = await replaceWith(async () => ({
          id: entityId,
          description_html: await confirm(),
          isMigrationUpdate: false,
        }));
        editorRef?.current?.setEditorValue(result.description_html, false);
      },
    }),
    [replaceWith, entityId, editorRef]
  );

  if (!workspaceDetails) return null;

  return (
    <>
      {/* Verification blocks input without rebuilding the editor, which would end its image transfers. */}
      <div className="contents" inert={verifying} aria-busy={verifying}>
        <RichTextEditor
          key={entityId}
          editable={!disabled && !replacing}
          ref={editorRef}
          images={images}
          id={entityId}
          issueSequenceId={issueSequenceId}
          disabledExtensions={disabledExtensions}
          initialValue={(disabled ? initialValue : localDescription.description_html) ?? "<p></p>"}
          value={swrDescription ?? null}
          workspaceSlug={workspaceSlug}
          workspaceId={workspaceDetails.id}
          projectId={projectId}
          dragDropEnabled
          onChange={(_document, description_html, options) => {
            change({ id: entityId, description_html, isMigrationUpdate: !!options?.isMigrationUpdate });
          }}
          placeholder={placeholder ?? ((isFocused, value) => t(getDescriptionPlaceholderI18n(isFocused, value)))}
          containerClassName={containerClassName}
        />
      </div>
      {unfinished && !disabled && <p role="status">{t("content_files.image.unfinished")}</p>}
      {imageFailure && <p role="alert">{t("content_files.image.unavailable")}</p>}
      {needsResume && !disabled && (
        <div inert={verifying}>
          <PreservedInput
            conflict={conflict}
            resume={() => void recover()}
            discard={() => {
              discard();
              editorRef.current?.setEditorValue(initialValue ?? "<p></p>", false);
            }}
          />
        </div>
      )}
      {needsResume && disabled && <p role="status">{t("content_files.input.read_only")}</p>}
      <AutosaveError failure={failure} retry={needsResume ? undefined : retry} />
    </>
  );
});
