/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState, useLayoutEffect, useCallback } from "react";
import { Command } from "cmdk";
import { observer } from "mobx-react";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { isComposing } from "@nerve/utils";
// hooks
import { usePowerK } from "@/hooks/store/use-power-k";
// local imports
import type { TPowerKCommandConfig, TPowerKContext } from "../../core/types";
import type { TPowerKCommandsListProps } from "./commands-list";
import { PowerKModalFooter } from "./footer";
import { PowerKModalHeader } from "./header";

type Props = {
  commandsListComponent: React.FC<TPowerKCommandsListProps>;
  context: TPowerKContext;
  isOpen: boolean;
  onClose: () => void;
};

export const ProjectsAppPowerKModalWrapper = observer(function ProjectsAppPowerKModalWrapper(props: Props) {
  const { commandsListComponent: CommandsListComponent, context, isOpen, onClose } = props;
  const { setActiveCommand, setShouldShowContextBasedActions } = context;
  // states
  const [searchTerm, setSearchTerm] = useState("");
  const [isWorkspaceLevel, setIsWorkspaceLevel] = useState(false);
  // store hooks
  const { activePage, setActivePage } = usePowerK();

  // Handle command selection
  const handleCommandSelect = useCallback(
    (command: TPowerKCommandConfig) => {
      if (!isOpen) return;
      if (command.type === "action") {
        // Direct action - execute and potentially close
        command.action(context);
        if (command.closeOnSelect === true) {
          context.closePalette();
        }
      } else if (command.type === "change-page") {
        // Opens a selection page
        context.setActiveCommand(command);
        setActivePage(command.page);
        setSearchTerm("");
      }
    },
    [isOpen, context, setActivePage]
  );

  // Handle selection page item selection
  const handlePageDataSelection = useCallback(
    (data: unknown) => {
      if (!isOpen) return;
      if (context.activeCommand?.type === "change-page") {
        context.activeCommand.onSelect(data, context);
      }
      // Go back to main page
      if (context.activeCommand?.closeOnSelect === true) {
        context.closePalette();
      }
    },
    [isOpen, context]
  );

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // a key an input method takes is the method's: an Escape that ends a composition in the search neither clears it
      // nor closes the palette, a Backspace in one leaves the page where it is
      // The exit animation can retain this input after close; only the open host owns its keys.
      if (!isOpen || isComposing(e.nativeEvent)) return;
      // Cmd/Ctrl+K closes palette
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        // This close must not also reach the document shortcut that opens the palette.
        e.stopPropagation();
        onClose();
        return;
      }

      // Escape closes palette or clears search
      if (e.key === "Escape") {
        e.preventDefault();
        if (searchTerm) {
          setSearchTerm("");
        } else {
          onClose();
        }
        return;
      }

      // Backspace clears context or goes back from page
      if (e.key === "Backspace" && !searchTerm) {
        e.preventDefault();
        if (activePage) {
          // Go back from selection page
          setActivePage(null);
          context.setActiveCommand(null);
        } else {
          // Hide context based actions
          context.setShouldShowContextBasedActions(false);
        }
        return;
      }
    },
    [isOpen, searchTerm, activePage, onClose, setActivePage, context]
  );

  // Clear this host when it closes; no timer from a previous opening may erase a new query or command.
  useLayoutEffect(() => {
    if (isOpen) return;
    setSearchTerm("");
    setActivePage(null);
    setActiveCommand(null);
    setShouldShowContextBasedActions(true);
  }, [isOpen, setActivePage, setActiveCommand, setShouldShowContextBasedActions]);

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={onClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
      className="divide-opacity-10 flex flex-col items-center justify-center divide-y divide-subtle-1"
    >
      <Command
        // An exiting list must release Ctrl+K to the document opener, not consume it as Vim navigation.
        vimBindings={isOpen}
        filter={(i18nValue: string, search: string) => {
          if (i18nValue === "no-results") return 1;
          if (i18nValue.toLowerCase().includes(search.toLowerCase())) return 1;
          return 0;
        }}
        shouldFilter={searchTerm.length > 0 && (!!activePage || !context.params.workspaceSlug)}
        onKeyDown={handleKeyDown}
        className="w-full"
      >
        <PowerKModalHeader
          activePage={activePage}
          context={context}
          onSearchChange={setSearchTerm}
          searchTerm={searchTerm}
        />
        <Command.List className="vertical-scrollbar scrollbar-sm max-h-96 overflow-scroll outline-none">
          <CommandsListComponent
            isOpen={isOpen}
            activePage={activePage}
            context={context}
            handleCommandSelect={handleCommandSelect}
            handlePageDataSelection={handlePageDataSelection}
            isWorkspaceLevel={isWorkspaceLevel}
            searchTerm={searchTerm}
            setSearchTerm={setSearchTerm}
          />
        </Command.List>
        {/* Footer hints */}
        <PowerKModalFooter
          isWorkspaceLevel={isWorkspaceLevel}
          projectId={context.params.projectId}
          onWorkspaceLevelChange={setIsWorkspaceLevel}
        />
      </Command>
    </ModalCore>
  );
});
