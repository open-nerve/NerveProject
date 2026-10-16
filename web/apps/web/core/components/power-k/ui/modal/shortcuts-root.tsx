/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { ScrollArea } from "@nerve/propel/scrollarea";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { usePowerK } from "@/hooks/store/use-power-k";
// local imports
import { ShortcutRenderer } from "../renderer/shortcut";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function ShortcutsModal(props: Props) {
  const { isOpen, onClose } = props;
  // states
  const [query, setQuery] = useState("");
  // store hooks
  const { commandRegistry } = usePowerK();
  const { t } = useTranslation();

  // Get all commands from registry
  const allCommandsWithShortcuts = commandRegistry.getAllCommandsWithShortcuts();

  const handleClose = () => {
    onClose();
    setQuery("");
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.MD}
      className="flex h-[61vh] flex-col space-y-4 overflow-hidden py-5"
    >
      {/* the title names the dialog alone: its close button stands beside it */}
      <div className="flex justify-between px-5">
        <ModalTitle className="text-16 font-medium">{t("keyboard_shortcuts")}</ModalTitle>
        <button type="button" onClick={handleClose} aria-label={t("close")}>
          <CloseOutline className="h-4 w-4 text-secondary hover:text-primary" aria-hidden="true" />
        </button>
      </div>
      <div className="px-5">
        <InputGroup size="2xl">
          <SearchOutline className="h-3.5 w-3.5 text-secondary" />
          <Input
            size="2xl"
            id="search"
            name="search"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("power_k.search_shortcuts")}
            aria-label={t("power_k.search_shortcuts")}
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- the dialog opens on its search, as the palette does
            autoFocus
          />
        </InputGroup>
      </div>

      <ScrollArea size="sm" rootClassName="overflow-y-scroll px-5">
        <ShortcutRenderer searchQuery={query} commands={allCommandsWithShortcuts} />
      </ScrollArea>
    </ModalCore>
  );
}
