import { CheckCircle2, Copy } from "lucide-react";
import { useState } from "react";

interface AddressDisplayProps {
  /** The Ethereum address to display */
  address: string;
  /** Label to show before the address (e.g., "To:", "Contract:") */
  label?: string;
  /** Custom class names for styling */
  className?: string;
  /** Show fewer characters in truncated view */
  compact?: boolean;
  /** Test ID for the component */
  testId?: string;
}

/**
 * A reusable component for displaying Ethereum addresses with copy functionality
 * Automatically truncates long addresses and provides a copy-to-clipboard button
 */
export function AddressDisplay({
  address,
  label,
  className = "",
  compact = false,
  testId,
}: AddressDisplayProps) {
  const [copiedAddress, setCopiedAddress] = useState<string | null>(null);

  /**
   * Copies the address to clipboard and shows visual feedback
   */
  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopiedAddress(address);
      setTimeout(() => setCopiedAddress(null), 2000);
    } catch (err) {
      console.error("Failed to copy address:", err);
    }
  };

  /**
   * Truncates the address for display
   * Compact mode: shows first 6 + last 4 characters
   * Normal mode: shows first 8 + last 6 characters
   */
  const getTruncatedAddress = () => {
    if (compact) {
      return `${address.slice(0, 6)}...${address.slice(-4)}`;
    }
    return `${address.slice(0, 8)}...${address.slice(-6)}`;
  };

  return (
    <div
      className={`flex items-center space-x-2 ${className}`}
      data-testid={testId}
    >
      {label && (
        <span className="text-xs text-gray-500 font-medium">{label}</span>
      )}
      <code
        title={address}
        className="text-xs font-mono bg-gray-100 px-2 py-1 rounded border text-gray-700 overflow-hidden whitespace-nowrap text-ellipsis max-w-32"
      >
        {getTruncatedAddress()}
      </code>
      <button
        onClick={copyToClipboard}
        className="p-1 hover:bg-gray-100 rounded transition-colors flex-shrink-0"
        title="Copy address"
        data-testid={testId ? `${testId}-copy-button` : undefined}
      >
        {copiedAddress === address ? (
          <CheckCircle2 className="h-3 w-3 text-green-500" />
        ) : (
          <Copy className="h-3 w-3 text-gray-500" />
        )}
      </button>
    </div>
  );
}
