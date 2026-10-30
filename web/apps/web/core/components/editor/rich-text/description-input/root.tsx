/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { AutosaveError, useConfirmedAutosave } from "../../use-confirmed-autosave";
import { observer } from "mobx-react";
// nerve imports
import type { EditorRefApi, TExtensions } from "@nerve/editor";
import { useTranslation } from "@nerve/i18n";
import type { EFileAssetType, TNameDescriptionLoader } from "@nerve/types";
import { getDescriptionPlaceholderI18n } from "@nerve/utils";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// hooks
import { useEditorAsset } from "@/hooks/store/use-editor-asset";
import { useWorkspace } from "@/hooks/store/use-workspace";
// local imports
type TFormData = {
  id: string;
  description_html: string;
  isMigrationUpdate: boolean;
};

type Props = {
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
   * @description File asset type, this will be used to upload the file to the editor
   */
  fileAssetType: EFileAssetType;
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
    containerClassName,
    disabled,
    disabledExtensions,
    editorRef,
    entityId,
    fileAssetType,
    initialValue,
    issueSequenceId,
    onSubmit,
    placeholder,
    projectId,
    setIsSubmitting,
    swrDescription,
    workspaceSlug,
  } = props;
  const { getWorkspaceBySlug } = useWorkspace();
  const { uploadEditorAsset, duplicateEditorAsset } = useEditorAsset();
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
  } = useConfirmedAutosave({
    initial,
    same: sameDescription,
    save: (input) => onSubmit({ description_html: input.description_html }, input.isMigrationUpdate),
    status: setIsSubmitting,
    disabled,
  });

  if (!workspaceDetails) return null;

  return (
    <>
      <RichTextEditor
        key={entityId}
        editable={!disabled}
        ref={editorRef}
        id={entityId}
        issueSequenceId={issueSequenceId}
        disabledExtensions={disabledExtensions}
        initialValue={localDescription.description_html ?? "<p></p>"}
        value={swrDescription ?? null}
        workspaceSlug={workspaceSlug}
        workspaceId={workspaceDetails.id}
        projectId={projectId}
        dragDropEnabled
        onChange={(_document, description_html, options) => {
          if (description_html === localDescription.description_html) return;
          change({ id: entityId, description_html, isMigrationUpdate: !!options?.isMigrationUpdate });
        }}
        placeholder={placeholder ?? ((isFocused, value) => t(getDescriptionPlaceholderI18n(isFocused, value)))}
        containerClassName={containerClassName}
        uploadFile={async (blockId, file) => {
          try {
            const { asset_id } = await uploadEditorAsset({
              blockId,
              data: {
                entity_identifier: entityId,
                entity_type: fileAssetType,
              },
              file,
              projectId,
              workspaceSlug,
            });
            return asset_id;
          } catch (error) {
            console.log("Error in uploading asset:", error);
            throw new Error("Asset upload failed. Please try again later.", { cause: error });
          }
        }}
        duplicateFile={async (assetId: string) => {
          try {
            const { asset_id } = await duplicateEditorAsset({
              assetId,
              entityType: fileAssetType,
              projectId,
              workspaceSlug,
            });
            return asset_id;
          } catch {
            throw new Error("Asset duplication failed. Please try again later.");
          }
        }}
      />
      <AutosaveError failure={failure} retry={retry} />
    </>
  );
});
