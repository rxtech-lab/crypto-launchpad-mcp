import { Info, FileText } from "lucide-react";
import type { TransactionMetadata } from "../types/wallet";

interface MetadataDisplayProps {
  metadata: TransactionMetadata[];
  sessionId?: string;
}

export function MetadataDisplay({ metadata, sessionId }: MetadataDisplayProps) {
  if (!metadata || metadata.length === 0) {
    return null;
  }

  return (
    <div 
      data-testid="metadata-container"
      className="bg-white border border-gray-200 rounded-lg p-4 animate-fade-in"
    >
      <div className="flex items-center space-x-2 mb-3">
        <Info className="h-5 w-5 text-blue-500" />
        <h3 className="text-lg font-semibold text-gray-800">
          Session Information
        </h3>
      </div>

      {sessionId && (
        <div className="mb-3 pb-3 border-b border-gray-100">
          <p 
            data-testid="session-id"
            className="text-xs text-gray-500 font-mono"
          >
            Session ID: {sessionId}
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {metadata.map((item, index) => (
          <div 
            key={index}
            data-testid={`metadata-item-${index}`}
            className="flex items-start space-x-2"
          >
            <FileText className="h-4 w-4 text-gray-400 mt-0.5 flex-shrink-0" />
            <div className="min-w-0 flex-grow">
              <p 
                data-testid={`metadata-title-${index}`}
                className="text-xs text-gray-500 uppercase tracking-wider"
              >
                {item.title}
              </p>
              <p
                data-testid={`metadata-value-${index}`}
                className="text-sm text-gray-700 font-medium truncate"
                title={item.value}
              >
                {item.description}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
