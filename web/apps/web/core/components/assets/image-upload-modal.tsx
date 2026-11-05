/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ProfileUploadTarget } from "@/store/asset/upload-policy";
import { useState, useEffect } from "react";
import { observer } from "mobx-react";
import { EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useProfileUpload } from "@/hooks/use-profile-upload";
import { ImagePicker } from "./image-picker";
import type { ImagePickerProps } from "./image-picker";

type Props = Omit<ImagePickerProps, "upload" | "onCancel" | "onSaved"> & {
  isOpen: boolean;
  title: string;
  target: ProfileUploadTarget;
  onClose(): void;
};
export const ImageUploadModal = observer(function ImageUploadModal(props: Props) {
  const { isOpen, title, target, onClose, onBusyChange, ...picker } = props;
  const upload = useProfileUpload(target, isOpen);
  const [operation, setOperation] = useState<{ upload: typeof upload; busy: boolean }>();
  const busy = !!picker.busyBinding || !!upload?.busy || (operation?.upload === upload && !!operation?.busy);
  const current = upload?.current;
  useEffect(() => {
    if (isOpen && upload && !current) onClose();
  }, [isOpen, upload, current, onClose]);
  const close = () => {
    if (!busy) onClose();
  };
  return (
    <ModalCore isOpen={isOpen && !!current} handleClose={busy ? undefined : close} width={EModalWidth.XL}>
      <div className="space-y-5 p-6">
        <ModalTitle className="text-16 font-medium text-primary">{title}</ModalTitle>
        {upload && (
          <ImagePicker
            {...picker}
            upload={upload}
            onCancel={close}
            onSaved={onClose}
            onBusyChange={(value) => {
              setOperation({ upload, busy: value });
              onBusyChange?.(value);
            }}
          />
        )}
      </div>
    </ModalCore>
  );
});
