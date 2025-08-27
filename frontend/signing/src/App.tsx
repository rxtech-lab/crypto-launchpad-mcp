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
          {wallet.chainId !== transaction.session?.chain_id && (
            <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
              <p className="text-sm text-amber-800">
                Wrong network detected. Please switch to the correct network
                when prompted.
              </p>
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
          <h2 className="text-xl font-semibold text-gray-800 mb-6">
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
      <div className="max-w-3xl mx-auto p-6">
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
                <div className="flex items-center space-x-2">
                  <Wallet className="w-4 h-4 text-gray-600" />
                  <span className="text-sm font-medium text-gray-700">
                    {formatAddress(wallet.account)}
                  </span>
                  <span className="text-sm text-gray-500">
                    • Chain ID: {wallet.chainId}
                  </span>
                </div>
                <button
                  onClick={wallet.disconnectWallet}
                  className="flex items-center space-x-1 px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded-md transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  <span>Disconnect</span>
                </button>
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
