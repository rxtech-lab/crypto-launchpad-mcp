import { useState } from "react";
import { XCircle, RefreshCw, ChevronDown, ChevronUp } from "lucide-react";

interface ErrorDisplayProps {
  error: Error | null;
  onRetry?: () => void;
}

export function ErrorDisplay({ error, onRetry }: ErrorDisplayProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  if (!error) return null;

  return (
    <div 
      data-testid="error-display-container"
      className="bg-red-50 border border-red-200 rounded-lg p-4 animate-fade-in"
    >
      <div className="flex items-start space-x-3">
        <XCircle className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
        <div className="flex-grow">
          <p className="text-sm font-medium text-red-800">Error Occurred</p>
          <p 
            data-testid="error-message"
            className="text-sm text-red-600 mt-1"
          >
            {error.message}
          </p>

          {error.stack && (
            <button
              data-testid="error-details-toggle"
              onClick={() => setIsExpanded(!isExpanded)}
              className="mt-2 flex items-center space-x-1 text-xs text-red-700 hover:text-red-800"
            >
              {isExpanded ? (
                <>
                  <ChevronUp className="h-3 w-3" />
                  <span>Hide Details</span>
                </>
              ) : (
                <>
                  <ChevronDown className="h-3 w-3" />
                  <span>Show Details</span>
                </>
              )}
            </button>
          )}

          {isExpanded && error.stack && (
            <pre 
              data-testid="error-stack-trace"
              className="mt-2 p-2 bg-red-100 rounded text-xs text-red-700 overflow-x-auto whitespace-pre-wrap break-all"
            >
              {error.stack}
            </pre>
          )}

          {onRetry && (
            <button
              data-testid="error-retry-button"
              onClick={onRetry}
              className="mt-3 flex items-center space-x-1 text-sm text-red-700 hover:text-red-800 font-medium"
            >
              <RefreshCw className="h-4 w-4" />
              <span>Retry</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
