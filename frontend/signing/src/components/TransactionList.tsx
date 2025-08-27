import {
  CheckCircle2,
  Clock,
  Copy,
  Layers,
  Loader2,
  XCircle,
} from "lucide-react";
import { useState } from "react";
import type { TransactionDeployment, TransactionStatus } from "../types/wallet";
import { formatEther } from "../utils/ethereum";

interface TransactionListProps {
  transactions: TransactionDeployment[];
  statuses: Map<number, TransactionStatus>;
  currentIndex: number;
  isExecuting: boolean;
  deployedContracts?: Map<number, { address: string; txHash: string }>;
}

export function TransactionList({
  transactions,
  statuses,
  currentIndex,
  isExecuting,
  deployedContracts,
}: TransactionListProps) {
  const [copiedAddress, setCopiedAddress] = useState<string | null>(null);

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedAddress(text);
      setTimeout(() => setCopiedAddress(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };
  const getStatusIcon = (
    status: TransactionStatus | undefined,
    index: number
  ) => {
    const isActive = isExecuting && index === currentIndex;

    switch (status) {
      case "confirmed":
        return <CheckCircle2 className="h-5 w-5 text-green-500" />;
      case "failed":
        return <XCircle className="h-5 w-5 text-red-500" />;
      case "pending":
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      case "waiting":
      default:
        return isActive ? (
          <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />
        ) : (
          <Clock className="h-5 w-5 text-gray-400" />
        );
    }
  };

  const getStatusColor = (status: TransactionStatus | undefined) => {
    switch (status) {
      case "confirmed":
        return "border-green-200 bg-green-50";
      case "failed":
        return "border-red-200 bg-red-50";
      case "pending":
        return "border-blue-200 bg-blue-50 animate-pulse-soft";
      case "waiting":
      default:
        return "border-gray-200 bg-white";
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
        const deployedContract = deployedContracts?.get(index);

        return (
          <div
            key={index}
            className={`
              p-4 rounded-lg border transition-all duration-300
              ${getStatusColor(status)}
              ${isActive ? "shadow-md animate-slide-up" : ""}
            `}
          >
            <div className="flex items-center">
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
                {status === "pending" && (
                  <p className="text-xs text-blue-600 mt-1">
                    Please confirm in your wallet...
                  </p>
                )}
              </div>

              <div className="text-right">
                <span className="font-mono text-sm text-gray-700">
                  {tx.value.length > 0 ? formatEther(tx.value) : "0"} ETH
                </span>
                {status === "confirmed" && (
                  <p className="text-xs text-green-600 mt-1">Confirmed</p>
                )}
                {status === "failed" && (
                  <p className="text-xs text-red-600 mt-1">Failed</p>
                )}
              </div>
            </div>

            {/* Show deployed contract address if available */}
            {status === "confirmed" && deployedContract && (
              <div className="mt-3 pt-3 border-t border-gray-200">
                <div className="flex items-center justify-between">
                  <span className="text-xs text-gray-500">
                    Contract Address:
                  </span>
                  <div className="flex items-center space-x-2">
                    <code className="text-xs font-mono bg-gray-100 px-2 py-1 rounded">
                      {deployedContract.address.slice(0, 6)}...
                      {deployedContract.address.slice(-4)}
                    </code>
                    <button
                      onClick={() => copyToClipboard(deployedContract.address)}
                      className="p-1 hover:bg-gray-100 rounded transition-colors"
                      title="Copy address"
                    >
                      {copiedAddress === deployedContract.address ? (
                        <CheckCircle2 className="h-3 w-3 text-green-500" />
                      ) : (
                        <Copy className="h-3 w-3 text-gray-500" />
                      )}
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
