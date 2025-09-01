"use client";

import React from "react";
import { Copy, CheckCheck } from "lucide-react";

interface CodeBlockProps {
  code: string;
  className?: string;
  onCopyStateChange?: (codeKey: string, copied: boolean) => void;
  copied?: boolean;
}

export function CodeBlock({ 
  code, 
  className = "bg-muted/50 rounded-lg font-mono text-sm overflow-x-auto max-w-3xl",
  onCopyStateChange,
  copied = false
}: CodeBlockProps) {
  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(code);
      const codeKey = code.substring(0, 50) + code.length;
      onCopyStateChange?.(codeKey, true);
      setTimeout(() => {
        onCopyStateChange?.(codeKey, false);
      }, 2000);
    } catch (err) {
      console.error("Failed to copy text: ", err);
    }
  };

  return (
    <div className={`${className} relative group`}>
      <pre className="p-4 pr-12">{code}</pre>
      <button
        onClick={copyToClipboard}
        className="absolute top-2 right-2 p-2 rounded-md bg-background/80 hover:bg-background border border-border/50 hover:border-border transition-all opacity-0 group-hover:opacity-100 focus:opacity-100"
        title={copied ? "Copied!" : "Copy to clipboard"}
      >
        {copied ? (
          <CheckCheck className="h-3.5 w-3.5 text-green-600" />
        ) : (
          <Copy className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" />
        )}
      </button>
    </div>
  );
}