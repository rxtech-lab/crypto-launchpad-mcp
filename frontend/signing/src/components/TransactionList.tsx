import { useState } from "react";
import {
  CheckCircle2,
  Clock,
  Layers,
  Loader2,
  XCircle,
  FileCode,
  Coins,
} from "lucide-react";
import type {
  TransactionDeployment,
  TransactionStatus,
  EIP6963Provider,
} from "../types/wallet";
import { formatEther } from "../utils/ethereum";
import { useTokenBalance } from "../hooks/useTokenBalance";
import { AddressDisplay } from "./AddressDisplay";
import { ContractCodeDialog } from "./ContractCodeDialog";
import { ContractArgumentsTooltip } from "./ContractArgumentsTooltip";

interface TransactionListProps {
  transactions: TransactionDeployment[];
  statuses: Map<number, TransactionStatus>;
  currentIndex: number;
  isExecuting: boolean;
  deployedContracts?: Map<number, { address: string; txHash: string }>;
  provider?: EIP6963Provider;
  walletAddress?: string;
  chainId?: number;
}

export function TransactionList({
  transactions,
  statuses,
  currentIndex,
  isExecuting,
  deployedContracts,
  provider,
  walletAddress,
  chainId,
}: TransactionListProps) {
  const [codeDialogOpen, setCodeDialogOpen] = useState(false);
  const [selectedContract, setSelectedContract] = useState<{
    code: string;
    title: string;
  } | null>(null);
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

  const handleViewContractCode = (contractCode: string, title: string) => {
    setSelectedContract({ code: contractCode, title });
    setCodeDialogOpen(true);
  };

  // Token Balance Component
  function TokenBalanceDisplay({
    contractAddress,
    label,
    className = "",
  }: {
    contractAddress: string;
    label: string;
    className?: string;
  }) {
    const {
      data: tokenData,
      isLoading,
      error,
    } = useTokenBalance({
      contractAddress,
      walletAddress,
      provider,
      chainId,
      enabled: !!contractAddress && !!walletAddress && !!provider,
    });

    if (!contractAddress || !walletAddress || !provider) return null;
    if (isLoading)
      return (
        <div className={`text-xs text-gray-500 ${className}`}>
          Loading balance...
        </div>
      );
    if (error)
      return (
        <div className={`text-xs text-red-500 ${className}`}>
          Error loading balance
        </div>
      );
    if (!tokenData) return null;

    return (
      <div className={`flex items-center gap-2 text-sm ${className}`}>
        <Coins className="h-4 w-4 text-gray-500" />
        <span className="text-gray-600">{label}</span>
        <span className="font-mono font-medium">
          {tokenData.formattedBalance} {tokenData.symbol}
        </span>
      </div>
    );
  }

  return (
    <div data-testid="transaction-list-container" className="space-y-3">
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

        const TransactionContent = (
          <div
            key={index}
            data-testid={`transaction-item-${index}`}
            className={`
              p-4 rounded-lg border transition-all duration-300
              ${getStatusColor(status)}
              ${isActive ? "shadow-md animate-slide-up" : ""}
            `}
          >
            <div className="flex items-start">
              <div
                data-testid={`transaction-status-icon-${index}`}
                className="flex-shrink-0 mr-4 mt-1"
              >
                {getStatusIcon(status, index)}
              </div>

              <div className="flex-grow min-w-0">
                <div className="flex items-start justify-between">
                  <div className="flex-grow">
                    <h4
                      data-testid={`transaction-title-${index}`}
                      className="font-medium text-gray-800"
                    >
                      {tx.title || `Transaction ${index + 1}`}
                    </h4>
                    {tx.description && (
                      <p
                        title={tx.description}
                        className="text-sm text-gray-600 mt-1 overflow-hidden whitespace-nowrap text-ellipsis"
                      >
                        {tx.description}
                      </p>
                    )}
                  </div>

                  {/* Action buttons */}
                  <div className="flex items-center gap-2 ml-4">
                    {/* Contract code viewer button */}
                    {tx.contractCode && (
                      <button
                        onClick={() =>
                          handleViewContractCode(
                            tx.contractCode!,
                            tx.title || `Transaction ${index + 1}`
                          )
                        }
                        className="p-2 text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                        title="View contract source code"
                      >
                        <FileCode className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                {tx.receiver && (
                  <AddressDisplay
                    address={tx.receiver}
                    label="To:"
                    className="mt-2"
                    testId={`transaction-receiver-${index}`}
                  />
                )}

                {/* Balance before deployment */}
                {tx.showBalanceBeforeDeployment && tx.contractAddress && (
                  <TokenBalanceDisplay
                    contractAddress={tx.contractAddress}
                    label="Balance before:"
                    className="mt-2"
                  />
                )}

                {status === "pending" && (
                  <p className="text-xs text-blue-600 mt-1">
                    Please confirm in your wallet...
                  </p>
                )}
              </div>

              <div
                data-testid={`transaction-value-${index}`}
                className="text-right ml-4"
              >
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
                <AddressDisplay
                  address={deployedContract.address}
                  label="Contract Address:"
                  compact={true}
                  testId={`deployed-contract-address-${index}`}
                />

                {/* Balance after deployment */}
                {tx.showBalanceAfterDeployment && (
                  <TokenBalanceDisplay
                    contractAddress={deployedContract.address}
                    label="Balance after:"
                    className="mt-2"
                  />
                )}
              </div>
            )}
          </div>
        );

        // Wrap with ContractArgumentsTooltip if rawContractArguments exists
        return tx.rawContractArguments ? (
          <ContractArgumentsTooltip
            key={index}
            rawContractArguments={tx.rawContractArguments}
          >
            {TransactionContent}
          </ContractArgumentsTooltip>
        ) : (
          TransactionContent
        );
      })}

      {/* Contract Code Dialog */}
      {selectedContract && (
        <ContractCodeDialog
          isOpen={codeDialogOpen}
          onClose={() => {
            setCodeDialogOpen(false);
            setSelectedContract(null);
          }}
          contractCode={selectedContract.code}
          title={selectedContract.title}
        />
      )}
    </div>
  );
}
