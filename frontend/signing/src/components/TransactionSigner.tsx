import { Send, Loader2, CheckCircle2, AlertCircle, RefreshCw } from 'lucide-react';

interface TransactionSignerProps {
  isExecuting: boolean;
  isConnected: boolean;
  hasTransactions: boolean;
  currentIndex: number;
  totalTransactions: number;
  error: Error | null;
  allCompleted: boolean;
  onSign: () => void;
  onRetry: () => void;
}

export function TransactionSigner({
  isExecuting,
  isConnected,
  hasTransactions,
  currentIndex,
  totalTransactions,
  error,
  allCompleted,
  onSign,
  onRetry
}: TransactionSignerProps) {
  const getButtonContent = () => {
    if (error) {
      return (
        <>
          <AlertCircle className="h-5 w-5" />
          <span>Transaction Failed</span>
        </>
      );
    }

    if (allCompleted) {
      return (
        <>
          <CheckCircle2 className="h-5 w-5" />
          <span>All Transactions Complete</span>
        </>
      );
    }

    if (isExecuting) {
      return (
        <>
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>
            Signing {currentIndex + 1} of {totalTransactions}...
          </span>
        </>
      );
    }

    return (
      <>
        <Send className="h-5 w-5" />
        <span>Sign & Send Transactions</span>
      </>
    );
  };

  const getButtonStyle = () => {
    if (error) return 'bg-red-600 hover:bg-red-700';
    if (allCompleted) return 'bg-green-600 hover:bg-green-700';
    if (isExecuting) return 'bg-blue-600';
    return 'bg-blue-600 hover:bg-blue-700';
  };

  const isDisabled = !isConnected || !hasTransactions || isExecuting || allCompleted;

  return (
    <div className="space-y-4">
      <button
        onClick={error ? onRetry : onSign}
        disabled={isDisabled && !error}
        className={`
          w-full flex items-center justify-center space-x-2 px-6 py-3
          text-white font-medium rounded-lg shadow-sm
          transition-all duration-200 transform
          ${getButtonStyle()}
          ${isDisabled && !error ? 'opacity-50 cursor-not-allowed' : 'hover:scale-[1.02] active:scale-[0.98]'}
        `}
      >
        {getButtonContent()}
      </button>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg animate-fade-in">
          <div className="flex items-start space-x-3">
            <AlertCircle className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
            <div className="flex-grow">
              <p className="text-sm font-medium text-red-800">Transaction Error</p>
              <p className="text-sm text-red-600 mt-1">{error.message}</p>
              <button
                onClick={onRetry}
                className="mt-3 flex items-center space-x-1 text-sm text-red-700 hover:text-red-800 font-medium"
              >
                <RefreshCw className="h-4 w-4" />
                <span>Retry Transaction</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {!isConnected && (
        <p className="text-sm text-gray-500 text-center">
          Connect your wallet to sign transactions
        </p>
      )}

      {isExecuting && (
        <div className="flex justify-center">
          <div className="flex items-center space-x-2 text-sm text-blue-600">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Processing transaction {currentIndex + 1} of {totalTransactions}</span>
          </div>
        </div>
      )}

      {allCompleted && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg animate-fade-in">
          <div className="flex items-center space-x-3">
            <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-green-800">Success!</p>
              <p className="text-sm text-green-600 mt-1">
                All {totalTransactions} transaction{totalTransactions > 1 ? 's' : ''} completed successfully
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}