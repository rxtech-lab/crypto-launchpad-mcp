"use client";

import React from "react";
import { Check } from "lucide-react";
import { StaggerItem } from "@/components/animated-container";
import { InstallationStep as InstallationStepType, InstallationType, IDEConfig } from "@/types/quick-start";
import { CodeBlock } from "./code-block";
import { IDEDropdown } from "./ide-dropdown";

interface InstallationStepProps {
  step: InstallationStepType;
  type: InstallationType;
  availableIDEs: IDEConfig[];
  selectedIDE: string;
  dropdownOpen: boolean;
  copyStates: Record<string, boolean>;
  onToggleDropdown: () => void;
  onSelectIDE: (ideId: string) => void;
  onCopyStateChange: (codeKey: string, copied: boolean) => void;
}

export function InstallationStep({
  step,
  type,
  availableIDEs,
  selectedIDE,
  dropdownOpen,
  copyStates,
  onToggleDropdown,
  onSelectIDE,
  onCopyStateChange,
}: InstallationStepProps) {
  const renderBenefits = (benefits: string[]) => (
    <div className="space-y-2">
      {benefits.map((benefit, index) => (
        <div key={index} className="flex items-center gap-2 text-sm">
          <Check className="h-4 w-4 text-primary" />
          <span>{benefit}</span>
        </div>
      ))}
    </div>
  );

  return (
    <StaggerItem key={step.step}>
      <div className="bg-background rounded-xl border p-6 shadow-sm">
        <div className="flex items-start gap-4">
          <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
            <span className="text-sm font-semibold text-primary">
              {step.step}
            </span>
          </div>
          <div className="flex-1">
            <h3 className="font-semibold mb-2">{step.title}</h3>
            <p className="text-muted-foreground mb-4">{step.description}</p>

            {step.code && (
              <CodeBlock
                code={step.code}
                copied={copyStates[step.code.substring(0, 50) + step.code.length] || false}
                onCopyStateChange={onCopyStateChange}
              />
            )}

            {step.hasIDETabs && (
              <IDEDropdown
                type={type}
                availableIDEs={availableIDEs}
                selectedIDE={selectedIDE}
                isOpen={dropdownOpen}
                onToggle={onToggleDropdown}
                onSelect={onSelectIDE}
                copyStates={copyStates}
                onCopyStateChange={onCopyStateChange}
              />
            )}

            {step.benefits && renderBenefits(step.benefits)}
          </div>
        </div>
      </div>
    </StaggerItem>
  );
}