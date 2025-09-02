import { Activity, CheckCircle, Wallet, LogOut } from "lucide-react";
import { useCallback, useMemo } from "react";
import "./App.css";
import { ErrorDisplay } from "./components/ErrorDisplay";
import { HorizontalStepper } from "./components/HorizontalStepper";
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
    const rpcNetwork = wallet.getRPCNetworkMetadata();
    if (rpcNetwork && wallet.chainId !== rpcNetwork.chain_id) {
      try {
        await wallet.switchNetwork(rpcNetwork.chain_id);
      } catch (error) {
        console.error("Failed to switch network:", error);
        return;
      }
    }

    try {
      await transaction.executeAllTransactions(wallet.signTransaction, wallet.signMessage);
    } catch (error) {
      console.error("Transaction failed:", error);
    }
  }, [wallet, transaction]);

  const handleRetry = useCallback(() => {
    transaction.reset();
    transaction.loadSession();
  }, [transaction]);

  const handleSwitchNetwork = useCallback(async () => {
    const rpcNetwork = wallet.getRPCNetworkMetadata();
    if (!rpcNetwork) return;

    try {
      await wallet.switchNetwork(Number(rpcNetwork.chain_id));
      // Force re-check after network switch attempt
      window.location.reload();
    } catch (error) {
      console.error("Failed to switch network:", error);
    }
  }, [wallet]);

  const allCompleted = useMemo(() => {
    if (!transaction.session) return false;
    const totalTx = transaction.session.transaction_deployments.length;
    const completedCount = Array.from(
      transaction.transactionStatuses.values()
    ).filter((status) => status === "confirmed").length;
    return totalTx > 0 && completedCount === totalTx;
  }, [transaction.session, transaction.transactionStatuses]);

  // Check for network mismatch - compare wallet chain with RPC network metadata
  const networkMismatch = useMemo(() => {
    if (!wallet.isConnected) return false;
    const rpcNetwork = wallet.getRPCNetworkMetadata();
    if (!rpcNetwork) return false;
    console.log(
      "wallet.chainId",
      wallet.chainId,
      "rpcNetwork.chain_id",
      rpcNetwork.chain_id,
      "type of wallet.chainId",
      typeof wallet.chainId,
      "type of rpcNetwork.chain_id",
      typeof rpcNetwork.chain_id
    );
    // Ensure both values are numbers for comparison
    const walletChainId = Number(wallet.chainId);
    const requiredChainId = Number(rpcNetwork.chain_id);
    return walletChainId !== requiredChainId;
  }, [wallet]);

  // Determine current step for stepper
  const currentStep = useMemo(() => {
    if (allCompleted) return 2;
    if (wallet.isConnected) return 1;
    return 0;
  }, [wallet.isConnected, allCompleted]);

  const steps = [
    {
      id: "connect",
      title: "Connect Wallet",
    },
    {
      id: "review-sign",
      title: "Review & Sign",
    },
    {
      id: "complete",
      title: "Complete",
    },
  ];

  // Format wallet address for display
  const formatAddress = (address: string) => {
    if (!address) return "";
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  // Render current step content
  const renderStepContent = () => {
    // Step 1: Connect Wallet
    if (currentStep === 0) {
      return (
        <>
          <h2 className="text-xl font-semibold text-gray-800 mb-6">
            Connect Your Wallet
          </h2>
          <p className="text-gray-600 mb-6">
            Please connect your wallet to proceed with the transaction signing
            process.
          </p>
          <WalletSelector
            providers={wallet.providers}
            selectedProvider={wallet.selectedProvider}
            isConnecting={wallet.isConnecting}
            isConnected={wallet.isConnected}
            onConnect={wallet.connectWallet}
            onDisconnect={wallet.disconnectWallet}
          />
          {wallet.error && (
            <div className="mt-4">
              <ErrorDisplay error={wallet.error} />
            </div>
          )}
        </>
      );
    }

    // Step 2: Review & Sign
    if (currentStep === 1) {
      return (
        <>
          <h2 className="text-xl font-semibold text-gray-800 mb-6">
            Review & Sign Transactions
          </h2>

          {/* Network Status */}
          {networkMismatch && wallet.getRPCNetworkMetadata() && (
            <div className="mb-6 flex items-center justify-between p-4 bg-gradient-to-r from-amber-50 to-orange-50 border border-amber-200 rounded-xl">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-white rounded-lg shadow-sm">
                  <Activity className="h-5 w-5 text-amber-600" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-700">
                      Network Mismatch
                    </span>
                    <div className="h-2 w-2 bg-amber-500 rounded-full animate-pulse"></div>
                  </div>
                  <div className="text-base font-semibold text-gray-900 mb-1">
                    Please switch networks to continue
                  </div>
                  <div className="text-sm text-amber-700">
                    Current: Chain ID {wallet.chainId} → Required: Chain ID{" "}
                    {wallet.getRPCNetworkMetadata()?.chain_id || "Unknown"}
                  </div>
                  {wallet.networkSwitchError && (
                    <p className="text-sm text-red-600 mt-1">
                      Error: {wallet.networkSwitchError.message}
                    </p>
                  )}
                </div>
              </div>
              <button
                onClick={handleSwitchNetwork}
                className="px-4 py-2 text-sm font-medium text-amber-700 bg-white border border-amber-200 rounded-lg hover:bg-amber-50 hover:border-amber-300 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:ring-offset-2 transition-all duration-200 flex items-center gap-2"
                disabled={false} // Could add loading state here if needed
              >
                <Activity className="h-4 w-4" />
                <span>Switch Network</span>
              </button>
            </div>
          )}

          {/* Metadata if available */}
          {transaction.session?.metadata &&
            transaction.session.metadata.length > 0 && (
              <div className="mb-6 p-4 bg-gray-50 rounded-lg">
                <MetadataDisplay
                  metadata={transaction.session.metadata}
                  sessionId={transaction.session.id}
                />
              </div>
            )}

          {/* Transaction List */}
          {transaction.session && (
            <div className="mb-6">
              <TransactionList
                transactions={transaction.session.transaction_deployments}
                statuses={transaction.transactionStatuses}
                currentIndex={transaction.currentIndex}
                isExecuting={transaction.isExecuting}
                deployedContracts={transaction.deployedContracts}
              />
            </div>
          )}

          {/* Sign button and status */}
          <TransactionSigner
            isExecuting={transaction.isExecuting}
            isConnected={wallet.isConnected}
            hasTransactions={
              (transaction.session?.transaction_deployments.length || 0) > 0
            }
            currentIndex={transaction.currentIndex}
            totalTransactions={
              transaction.session?.transaction_deployments.length || 0
            }
            error={transaction.error}
            allCompleted={allCompleted}
            networkMismatch={networkMismatch}
            onSign={handleSignTransactions}
            onRetry={handleRetry}
          />
        </>
      );
    }

    // Step 3: Complete
    if (currentStep === 2) {
      return (
        <>
          <h2
            className="text-xl font-semibold text-gray-800 mb-6"
            data-testid="transaction-success-message"
          >
            All Transactions Complete
          </h2>

          <div className="text-center py-8">
            <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
            <p className="text-lg text-gray-700 mb-2">
              All transactions have been successfully executed!
            </p>
            <p className="text-sm text-gray-500">
              You can now close this window or view the transaction details
              below.
            </p>
          </div>

          {/* Final transaction list */}
          {transaction.session && (
            <div className="mt-6">
              <TransactionList
                transactions={transaction.session.transaction_deployments}
                statuses={transaction.transactionStatuses}
                currentIndex={transaction.currentIndex}
                isExecuting={false}
                deployedContracts={transaction.deployedContracts}
              />
            </div>
          )}
        </>
      );
    }

    return null;
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-3xl mx-auto p-6" data-testid="content-container">
        {/* Header */}
        <header className="mb-8 text-center">
          <div className="flex items-center justify-center space-x-3 mb-2">
            <Activity className="h-8 w-8 text-blue-600" />
            <h1 className="text-3xl font-bold text-gray-900">
              Transaction Signing
            </h1>
          </div>
          <p className="text-gray-600">
            Follow the steps to sign and execute your transactions
          </p>
        </header>

        {/* Main Card */}
        <div className="bg-white rounded-lg border border-gray-200">
          {/* Wallet Status Bar - Only show when connected */}
          {wallet.isConnected && wallet.account && (
            <div className="px-6 py-3 border-b border-gray-200 bg-gray-50">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <Wallet className="w-4 h-4 text-gray-600" />
                  <div className="flex flex-col">
                    <span className="text-sm font-medium text-gray-700">
                      {formatAddress(wallet.account)}
                    </span>
                    <span className="text-xs text-gray-500">
                      Chain ID: {wallet.chainId}
                    </span>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <button
                    onClick={wallet.disconnectWallet}
                    className="flex items-center space-x-1 px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded-md transition-colors"
                  >
                    <LogOut className="w-4 h-4" />
                    <span>Disconnect</span>
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Stepper */}
          <div className="px-6 py-4 border-b border-gray-200">
            <HorizontalStepper steps={steps} currentStep={currentStep} />
          </div>

          {/* Content */}
          <div className="p-6">
            {transaction.error && !transaction.session ? (
              <ErrorDisplay error={transaction.error} onRetry={handleRetry} />
            ) : !wallet.isConnected ? (
              // Always show wallet connection step when not connected
              renderStepContent()
            ) : !transaction.session && !transaction.error ? (
              // Show loading when wallet is connected but no session yet
              <div className="text-center py-8">
                <Activity className="h-12 w-12 text-gray-300 mx-auto mb-3 animate-pulse" />
                <p className="text-gray-500">Loading transaction session...</p>
              </div>
            ) : (
              renderStepContent()
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
