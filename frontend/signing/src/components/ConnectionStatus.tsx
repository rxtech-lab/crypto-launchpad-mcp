import React from 'react';
import { CheckCircle2, AlertTriangle, Link } from 'lucide-react';
import { formatAddress, getChainName } from '../utils/ethereum';

interface ConnectionStatusProps {
  isConnected: boolean;
  account: string | null;
  chainId: number | null;
  requiredChainId?: number;
}

export function ConnectionStatus({
  isConnected,
  account,
  chainId,
  requiredChainId
}: ConnectionStatusProps) {
  const isCorrectChain = !requiredChainId || chainId === requiredChainId;

  if (!isConnected) {
    return (
      <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg animate-fade-in">
        <div className="flex items-center space-x-3">
          <AlertTriangle className="h-5 w-5 text-amber-500 flex-shrink-0" />
          <div>
            <p className="text-sm font-medium text-amber-800">Wallet Not Connected</p>
            <p className="text-sm text-amber-600 mt-1">
              Please connect your wallet to continue
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3 animate-fade-in">
      <div className="p-4 bg-green-50 border border-green-200 rounded-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-green-800">Wallet Connected</p>
              <p className="text-sm text-green-600 mt-1 font-mono">
                {formatAddress(account || '')}
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-2 text-sm text-gray-600">
            <Link className="h-4 w-4" />
            <span>{getChainName(chainId || 0)}</span>
          </div>
        </div>
      </div>

      {requiredChainId && !isCorrectChain && (
        <div className="p-4 bg-amber-50 border border-amber-200 rounded-lg">
          <div className="flex items-center space-x-3">
            <AlertTriangle className="h-5 w-5 text-amber-500 flex-shrink-0" />
            <div>
              <p className="text-sm font-medium text-amber-800">Wrong Network</p>
              <p className="text-sm text-amber-600 mt-1">
                Please switch to {getChainName(requiredChainId)}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}