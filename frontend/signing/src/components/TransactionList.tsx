import { CheckCircle2, XCircle, Loader2, Clock, Layers } from 'lucide-react';
import type { TransactionDeployment, TransactionStatus } from '../types/wallet';
import { formatEther } from '../utils/ethereum';

interface TransactionListProps {
  transactions: TransactionDeployment[];
  statuses: Map<number, TransactionStatus>;
  currentIndex: number;
  isExecuting: boolean;
}

export function TransactionList({
  transactions,
  statuses,
  currentIndex,
  isExecuting
}: TransactionListProps) {
  const getStatusIcon = (status: TransactionStatus | undefined, index: number) => {
    const isActive = isExecuting && index === currentIndex;
    
    switch (status) {
      case 'confirmed':
        return <CheckCircle2 className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <XCircle className="h-5 w-5 text-red-500" />;
      case 'pending':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'waiting':
      default:
        return isActive 
          ? <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />
          : <Clock className="h-5 w-5 text-gray-400" />;
    }
  };

  const getStatusColor = (status: TransactionStatus | undefined) => {
    switch (status) {
      case 'confirmed':
        return 'border-green-200 bg-green-50';
      case 'failed':
        return 'border-red-200 bg-red-50';
      case 'pending':
        return 'border-blue-200 bg-blue-50 animate-pulse-soft';
      case 'waiting':
      default:
        return 'border-gray-200 bg-white';
    }
  };

  if (transactions.length === 0) {
    return (
      <div className="p-8 text-center text-gray-500">
        <Layers className="h-12 w-12 text-gray-300 mx-auto mb-3" />
        <p>No transactions to display</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800 flex items-center">
          <Layers className="h-5 w-5 mr-2 text-gray-600" />
          Transactions ({transactions.length})
        </h3>
        {isExecuting && (
          <span className="text-sm text-blue-600 font-medium">
            Processing {currentIndex + 1} of {transactions.length}
          </span>
        )}
      </div>

      {transactions.map((tx, index) => {
        const status = statuses.get(index);
        const isActive = isExecuting && index === currentIndex;
        
        return (
          <div
            key={index}
            className={`
              flex items-center p-4 rounded-lg border transition-all duration-300
              ${getStatusColor(status)}
              ${isActive ? 'shadow-md animate-slide-up' : ''}
            `}
          >
            <div className="flex-shrink-0 mr-4">
              {getStatusIcon(status, index)}
            </div>
            
            <div className="flex-grow">
              <h4 className="font-medium text-gray-800">
                {tx.title || `Transaction ${index + 1}`}
              </h4>
              {tx.description && (
                <p className="text-sm text-gray-600 mt-1">{tx.description}</p>
              )}
              {status === 'pending' && (
                <p className="text-xs text-blue-600 mt-1">
                  Please confirm in your wallet...
                </p>
              )}
            </div>
            
            <div className="text-right">
              <span className="font-mono text-sm text-gray-700">
                {formatEther(tx.value)} ETH
              </span>
              {status === 'confirmed' && (
                <p className="text-xs text-green-600 mt-1">Confirmed</p>
              )}
              {status === 'failed' && (
                <p className="text-xs text-red-600 mt-1">Failed</p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}