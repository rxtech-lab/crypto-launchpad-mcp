"use client";

import React from "react";
import { Check, ChevronDown, Terminal } from "lucide-react";
import { IDEConfig, InstallationType } from "@/types/quick-start";
import { CodeBlock } from "./code-block";

interface IDEDropdownProps {
  type: InstallationType;
  availableIDEs: IDEConfig[];
  selectedIDE: string;
  isOpen: boolean;
  onToggle: () => void;
  onSelect: (ideId: string) => void;
  copyStates: Record<string, boolean>;
  onCopyStateChange: (codeKey: string, copied: boolean) => void;
}

export function IDEDropdown({
  type,
  availableIDEs,
  selectedIDE,
  isOpen,
  onToggle,
  onSelect,
  copyStates,
  onCopyStateChange,
}: IDEDropdownProps) {
  const selectedIDEConfig = availableIDEs.find(ide => ide.id === selectedIDE);
  const SelectedIcon = selectedIDEConfig?.icon || Terminal;

  const renderIDEConfig = (ide: IDEConfig) => {
    const config = type === "local" ? ide.local : ide.remote;
    if (!config) return null;

    return (
      <div className="space-y-3 mt-4" key={ide.id}>
        <p className="text-sm text-muted-foreground">{config.description}</p>

        {"command" in config && config.command ? (
          <CodeBlock
            code={config.command}
            copied={copyStates[config.command.substring(0, 50) + config.command.length] || false}
            onCopyStateChange={onCopyStateChange}
          />
        ) : config.code ? (
          <CodeBlock
            code={config.code}
            copied={copyStates[config.code.substring(0, 50) + config.code.length] || false}
            onCopyStateChange={onCopyStateChange}
          />
        ) : null}

        {"alternativeDescription" in config && config.alternativeDescription && (
          <>
            <p className="text-xs text-muted-foreground">
              {config.alternativeDescription}
            </p>
            {"alternativeCode" in config && config.alternativeCode && (
              <CodeBlock
                code={config.alternativeCode}
                copied={copyStates[config.alternativeCode.substring(0, 50) + config.alternativeCode.length] || false}
                onCopyStateChange={onCopyStateChange}
              />
            )}
          </>
        )}

        {config.note && (
          <p className="text-xs text-muted-foreground">{config.note}</p>
        )}
      </div>
    );
  };

  return (
    <div className="w-full space-y-4">
      {/* IDE Dropdown */}
      <div id={`dropdown-container-${type}`} className="relative">
        <button
          type="button"
          className="relative w-full cursor-pointer rounded-lg bg-background border border-input px-3 py-2 text-left shadow-sm hover:border-ring focus:outline-none focus:border-ring sm:text-sm transition-colors"
          onClick={onToggle}
        >
          <span className="flex items-center">
            <SelectedIcon className="h-4 w-4 mr-2 text-muted-foreground" />
            <span className="block truncate">
              {selectedIDEConfig?.name || "Select IDE"}
            </span>
          </span>
          <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
            <ChevronDown
              className={`h-4 w-4 text-muted-foreground transition-transform ${
                isOpen ? "rotate-180" : ""
              }`}
            />
          </span>
        </button>

        {isOpen && (
          <div className="absolute z-10 mt-1 w-full max-h-60 overflow-auto rounded-lg bg-background border border-input shadow-lg focus:outline-none animate-in fade-in-0 zoom-in-95">
            <div className="py-1">
              {availableIDEs.map((ide) => (
                <button
                  key={ide.id}
                  type="button"
                  className="group relative w-full cursor-pointer select-none py-2 px-3 text-left hover:bg-muted focus:bg-muted focus:outline-none transition-colors"
                  onClick={() => onSelect(ide.id)}
                >
                  <div className="flex items-center">
                    <ide.icon className="h-4 w-4 mr-2 text-muted-foreground" />
                    <span
                      className={`block truncate ${
                        selectedIDE === ide.id
                          ? "font-semibold text-primary"
                          : "font-normal"
                      }`}
                    >
                      {ide.name}
                    </span>
                    {selectedIDE === ide.id && (
                      <Check className="h-4 w-4 ml-auto text-primary" />
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Selected IDE Configuration */}
      {selectedIDEConfig && renderIDEConfig(selectedIDEConfig)}
    </div>
  );
}