import { Activity } from "lucide-react";
import { useCallback, useMemo } from "react";
import "./App.css";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { ErrorDisplay } from "./components/ErrorDisplay";
import { MetadataDisplay } from "./components/MetadataDisplay";
import { TransactionList } from "./components/TransactionList";
import { TransactionSigner } from "./components/TransactionSigner";
import { WalletSelector } from "./components/WalletSelector";
import { useTransaction } from "./hooks/useTransaction";
import { useWallet } from "./hooks/useWallet";

function App() {
  const wallet = useWallet();
  const transaction = useTransaction();

  const handleSignTransactions = useCallback(async () => {
    if (!wallet.isConnected) {
      console.error("Wallet not connected");
      return;
    }

    // Check if we need to switch networks
    if (
      transaction.session?.chain_id &&
      wallet.chainId !== transaction.session.chain_id
    ) {
      try {
        await wallet.switchNetwork(transaction.session.chain_id);
      } catch (error) {
        console.error("Failed to switch network:", error);
        return;
      }
    }

    try {
      await transaction.executeAllTransactions(wallet.signTransaction);
    } catch (error) {
      console.error("Transaction failed:", error);
    }
  }, [wallet, transaction]);

  const handleRetry = useCallback(() => {
    transaction.reset();
    transaction.loadSession();
  }, [transaction]);

  const allCompleted = useMemo(() => {
    if (!transaction.session) return false;
    const totalTx = transaction.session.transaction_deployments.length;
    const completedCount = Array.from(
      transaction.transactionStatuses.values()
    ).filter((status) => status === "confirmed").length;
    return totalTx > 0 && completedCount === totalTx;
  }, [transaction.session, transaction.transactionStatuses]);

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto p-6">
        <header className="mb-8">
          <div className="flex items-center space-x-3 mb-2">
            <Activity className="h-8 w-8 text-blue-600" />
            <h1 className="text-3xl font-bold text-gray-900">
              Transaction Signing
            </h1>
          </div>
          <p className="text-gray-600">
            Connect your wallet and sign the required transactions
          </p>
        </header>

        <div className="space-y-6">
          <div className="bg-white rounded-xl shadow-sm p-6">
            <h2 className="text-lg font-semibold text-gray-800 mb-4">
              Wallet Connection
            </h2>
            <WalletSelector
              providers={wallet.providers}
              selectedProvider={wallet.selectedProvider}
              isConnecting={wallet.isConnecting}
              isConnected={wallet.isConnected}
              onConnect={wallet.connectWallet}
              onDisconnect={wallet.disconnectWallet}
            />
            {(wallet.isConnected || wallet.error) && (
              <div className="mt-4">
                <ConnectionStatus
                  isConnected={wallet.isConnected}
                  account={wallet.account}
                  chainId={wallet.chainId}
                  requiredChainId={transaction.session?.chain_id}
                />
              </div>
            )}
            {wallet.error && (
              <div className="mt-4">
                <ErrorDisplay error={wallet.error} />
              </div>
            )}
          </div>

          {transaction.session && (
            <>
              {transaction.session.metadata &&
                transaction.session.metadata.length > 0 && (
                  <div className="bg-white rounded-xl shadow-sm p-6">
                    <MetadataDisplay
                      metadata={transaction.session.metadata}
                      sessionId={transaction.session.id}
                    />
                  </div>
                )}

              <div className="bg-white rounded-xl shadow-sm p-6">
                <TransactionList
                  transactions={transaction.session.transaction_deployments}
                  statuses={transaction.transactionStatuses}
                  currentIndex={transaction.currentIndex}
                  isExecuting={transaction.isExecuting}
                />
              </div>

              <div className="bg-white rounded-xl shadow-sm p-6">
                <TransactionSigner
                  isExecuting={transaction.isExecuting}
                  isConnected={wallet.isConnected}
                  hasTransactions={
                    transaction.session.transaction_deployments.length > 0
                  }
                  currentIndex={transaction.currentIndex}
                  totalTransactions={
                    transaction.session.transaction_deployments.length
                  }
                  error={transaction.error}
                  allCompleted={allCompleted}
                  onSign={handleSignTransactions}
                  onRetry={handleRetry}
                />
              </div>
            </>
          )}

          {transaction.error && !transaction.session && (
            <div className="bg-white rounded-xl shadow-sm p-6">
              <ErrorDisplay error={transaction.error} onRetry={handleRetry} />
            </div>
          )}

          {!transaction.session && !transaction.error && (
            <div className="bg-white rounded-xl shadow-sm p-6 text-center text-gray-500">
              <Activity className="h-12 w-12 text-gray-300 mx-auto mb-3" />
              <p>Loading transaction session...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;
