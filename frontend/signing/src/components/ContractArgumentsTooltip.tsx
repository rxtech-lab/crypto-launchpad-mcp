/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from "react";
import { Settings, Copy } from "lucide-react";

interface ContractArgument {
  name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value: any;
  type?: string;
}

interface ContractArgumentsTooltipProps {
  rawContractArguments: string;
  children: React.ReactNode;
}

export function ContractArgumentsTooltip({
  rawContractArguments,
  children,
}: ContractArgumentsTooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [arguments_, setArguments] = useState<ContractArgument[]>([]);

  // Parse contract arguments when component mounts or arguments change
  const parseArguments = (raw: string): ContractArgument[] => {
    try {
      const parsed = JSON.parse(raw);

      // If it's an array of values, create generic names
      if (Array.isArray(parsed)) {
        return parsed.map((value, index) => ({
          name: `arg${index}`,
          value: formatArgumentValue(value),
          type: typeof value,
        }));
      }

      // If it's an object with key-value pairs
      if (typeof parsed === "object" && parsed !== null) {
        return Object.entries(parsed).map(([name, value]) => ({
          name,
          value: formatArgumentValue(value),
          type: typeof value,
        }));
      }

      // Single value
      return [
        {
          name: "value",
          value: formatArgumentValue(parsed),
          type: typeof parsed,
        },
      ];
    } catch (error) {
      console.error("Failed to parse contract arguments:", error);
      return [
        {
          name: "raw",
          value: raw,
          type: "string",
        },
      ];
    }
  };

  const formatArgumentValue = (value: any): string => {
    if (value === null || value === undefined) return "null";
    if (typeof value === "string") return value;
    if (typeof value === "boolean") return value.toString();
    if (typeof value === "number") return value.toString();
    if (typeof value === "bigint") return value.toString();
    if (Array.isArray(value))
      return `[${value.map(formatArgumentValue).join(", ")}]`;
    if (typeof value === "object") return JSON.stringify(value, null, 2);
    return String(value);
  };

  const handleCopyArguments = async () => {
    try {
      await navigator.clipboard.writeText(rawContractArguments);
    } catch (error) {
      console.error("Failed to copy arguments:", error);
    }
  };

  const handleMouseEnter = () => {
    if (rawContractArguments) {
      setArguments(parseArguments(rawContractArguments));
      setIsVisible(true);
    }
  };

  const handleMouseLeave = () => {
    setIsVisible(false);
  };

  if (!rawContractArguments) {
    return <>{children}</>;
  }

  return (
    <div className="relative">
      <div
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        className="cursor-help"
      >
        {children}
        {/* Hover indicator */}
        <div className="absolute top-2 right-2 opacity-60 hover:opacity-100 transition-opacity">
          <Settings className="h-4 w-4 text-gray-500" />
        </div>
      </div>

      {/* Tooltip */}
      {isVisible && (
        <div
          className="absolute z-50 w-80 p-4 bg-white border border-gray-200 rounded-lg shadow-lg"
          style={{
            bottom: "100%",
            left: "50%",
            transform: "translateX(-50%)",
            marginBottom: "8px",
          }}
          onMouseEnter={() => setIsVisible(true)}
          onMouseLeave={handleMouseLeave}
        >
          {/* Tooltip arrow */}
          <div
            className="absolute w-3 h-3 bg-white border-r border-b border-gray-200 transform rotate-45"
            style={{
              bottom: "-6px",
              left: "50%",
              marginLeft: "-6px",
            }}
          />

          <div className="flex items-center justify-between mb-3">
            <h4 className="font-semibold text-gray-900 text-sm">
              Constructor Arguments
            </h4>
            <button
              onClick={handleCopyArguments}
              className="p-1 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded transition-colors"
              title="Copy raw arguments"
            >
              <Copy className="h-4 w-4" />
            </button>
          </div>

          <div className="space-y-2 max-h-60 overflow-y-auto">
            {arguments_.map((arg, index) => (
              <div key={index} className="bg-gray-50 rounded-md p-2">
                <div className="flex items-center justify-between mb-1">
                  <span className="font-medium text-xs text-gray-700">
                    {arg.name}
                  </span>
                  {arg.type && (
                    <span className="text-xs text-gray-500 bg-gray-200 px-1.5 py-0.5 rounded">
                      {arg.type}
                    </span>
                  )}
                </div>
                <div className="text-sm text-gray-900 font-mono break-all">
                  {arg.value}
                </div>
              </div>
            ))}
          </div>

          {arguments_.length === 0 && (
            <div className="text-sm text-gray-500 text-center py-2">
              No arguments found
            </div>
          )}
        </div>
      )}
    </div>
  );
}
