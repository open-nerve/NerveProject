/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
import { CoverImage } from "@/components/common/cover-image";
import ProjectCommonAttributes from "@/components/project/create/common-attributes";
import ProjectCreateHeader from "@/components/project/create/header";
import ProjectCreateButtons from "@/components/project/create/project-create-buttons";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useProfileUpload } from "@/hooks/use-profile-upload";
import useKeypress from "@/hooks/use-keypress";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { projectCoverURL } from "@/helpers/project-covers";
import { useRefusedFocus } from "@/lib/refused-focus";
import { ProjectAttributes } from "./attributes";
import { CreateProjectCover } from "./cover";
import { useCreateProject, type ProjectCreationForm } from "./use-create-project";
import { getProjectFormValues } from "./utils";

export type TCreateProjectFormProps = {
  workspaceSlug: string;
  onClose: () => void;
  handleNextStep: (projectId: string) => void;
};

export const CreateProjectForm = observer(function CreateProjectForm(props: TCreateProjectFormProps) {
  const workspace = useWorkspace().getWorkspaceBySlug(props.workspaceSlug);
  return <ProjectCreationFields key={workspace?.id ?? props.workspaceSlug} {...props} workspaceID={workspace?.id} />;
});

const ProjectCreationFields = observer(function ProjectCreationFields({
  workspaceSlug,
  workspaceID,
  onClose,
  handleNextStep,
}: TCreateProjectFormProps & { workspaceID?: string }) {
  const create = useCreateProject();
  const upload = useProfileUpload(
    { purpose: "PROJECT_COVER", scope: { kind: "new_project", id: workspaceID ?? "" } },
    !!workspaceID
  );
  const [shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier] = useState(true);
  const [defaults] = useState(getProjectFormValues);
  const [coverOpen, setCoverOpen] = useState(false);
  const [coverBusy, setCoverBusy] = useState(false);
  const [creating, setCreating] = useState(false);
  const coverButton = useRef<HTMLButtonElement>(null);
  const returnFocus = useRef(false);
  const pending = useRef(false);
  const coverPending = useRef(false);
  const life = useRef<{ mounted: boolean; upload: typeof upload }>({ mounted: false, upload });
  useLayoutEffect(() => {
    life.current = { mounted: true, upload };
    return () => {
      life.current.mounted = false;
    };
  }, [upload]);
  const methods = useForm<ProjectCreationForm>({
    defaultValues: defaults,
    reValidateMode: "onChange",
    shouldFocusError: false,
  });
  const {
    handleSubmit,
    setError,
    setValue,
    getValues,
    watch,
    formState: { isSubmitting, submitCount, errors },
  } = methods;
  const { isMobile } = usePlatformOS();
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const busy = creating || isSubmitting || coverBusy || !!upload?.busy;
  const canCreate = () =>
    !!upload?.current &&
    !["selected", "preparing", "uploading", "confirming", "failed"].includes(upload.phase) &&
    (!getValues("cover_image_asset_id") || getValues("cover_image_asset_id") === upload.readyAsset?.id);
  const canSubmit = canCreate();
  const assetID = watch("cover_image_asset_id");
  const coverURL =
    projectCoverURL(watch("cover_builtin_id")) ??
    (assetID && assetID === upload?.readyAsset?.id ? upload?.previewURL : undefined);
  const current = useCallback(
    () => life.current.mounted && life.current.upload === upload && !!upload?.current,
    [upload]
  );
  useLayoutEffect(() => {
    if (coverOpen) returnFocus.current = true;
    else if (!busy && returnFocus.current) {
      returnFocus.current = false;
      if (current()) coverButton.current?.focus({ preventScroll: true });
    }
  }, [coverOpen, busy, current, upload?.current]);
  const setCoverBusyState = (value: boolean) => {
    coverPending.current = value;
    setCoverBusy(value);
  };
  useEffect(() => {
    if (upload && !upload.current) onClose();
  }, [upload, upload?.current, onClose]);

  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (pending.current || coverPending.current || busy || !canCreate() || !current()) return;
    // Acquire before react-hook-form's asynchronous validation, so two native submits cannot queue two POSTs.
    pending.current = true;
    setCreating(true);
    void handleSubmit(async (values) => {
      if (!current() || !canCreate()) return;
      const project = await create(workspaceSlug, values, setError, current);
      if (project && current()) handleNextStep(project.id);
    })(event).finally(() => {
      if (current()) {
        pending.current = false;
        setCreating(false);
      }
    });
  };
  const handleClose = () => {
    if (pending.current || coverPending.current || busy) return;
    if (life.current.mounted) {
      life.current.mounted = false;
      upload?.dispose();
      onClose();
    }
  };
  useKeypress("Escape", () => {
    if (pending.current || coverPending.current || busy) return;
    if (coverOpen) {
      upload?.cancel();
      if (getValues("cover_image_asset_id")) setValue("cover_image_asset_id", undefined, { shouldDirty: true });
      setCoverOpen(false);
    } else handleClose();
  });

  return (
    <FormProvider {...methods}>
      <ProjectCreateHeader
        handleClose={handleClose}
        coverButtonRef={coverButton}
        onEditCover={() => setCoverOpen(true)}
        coverURL={coverURL}
        busy={busy}
        canEditCover={!!upload?.current}
      />
      <form ref={formRef} onSubmit={onSubmit} className="px-3">
        <div className="mt-9 space-y-6 pb-5">
          {errors.root?.creation?.message && (
            <p role="alert" className="text-body-sm-regular text-danger-primary">
              {errors.root.creation.message}
            </p>
          )}
          {coverOpen && upload && upload.current && (
            <CreateProjectCover
              upload={upload}
              busyBinding={creating || isSubmitting}
              onBusyChange={setCoverBusyState}
              onClose={() => setCoverOpen(false)}
              currentImage={<CoverImage src={coverURL} showDefaultWhenEmpty className="max-h-52 rounded-md" />}
            />
          )}
          <fieldset disabled={creating || isSubmitting} className="space-y-6">
            <ProjectCommonAttributes
              setValue={setValue}
              isMobile={isMobile}
              shouldAutoSyncIdentifier={shouldAutoSyncIdentifier}
              setShouldAutoSyncIdentifier={setShouldAutoSyncIdentifier}
            />
            <ProjectAttributes />
          </fieldset>
        </div>
        <ProjectCreateButtons handleClose={handleClose} busy={busy} canSubmit={canSubmit} />
      </form>
    </FormProvider>
  );
});
