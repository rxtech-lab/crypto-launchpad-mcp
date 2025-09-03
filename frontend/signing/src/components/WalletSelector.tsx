import { useEffect } from "react";
import { Loader2, Wallet, ChevronDown, LogOut, CheckCircle } from "lucide-react";
import type { EIP6963Provider } from "../types/wallet";

interface WalletSelectorProps {
  providers: EIP6963Provider[];
  selectedProvider: EIP6963Provider | null;
  isConnecting: boolean;
  isConnected: boolean;
  onConnect: (providerUuid: string) => void;
  onDisconnect?: () => void;
}

const WALLET_STORAGE_KEY = "selectedWalletUuid";

export function WalletSelector({
  providers,
  selectedProvider,
  isConnecting,
  isConnected,
  onConnect,
  onDisconnect,
}: WalletSelectorProps) {

  // Save selected wallet to localStorage when connected
  useEffect(() => {
    if (isConnected && selectedProvider) {
      localStorage.setItem(WALLET_STORAGE_KEY, selectedProvider.info.uuid);
    }
  }, [isConnected, selectedProvider]);

  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const providerUuid = event.target.value;
    if (providerUuid && !isConnected) {
      const provider = providers.find((p) => p.info.uuid === providerUuid);
      if (provider) {
        onConnect(provider.info.uuid);
      }
    }
  };

  const handleDisconnect = () => {
    localStorage.removeItem(WALLET_STORAGE_KEY);
    if (onDisconnect) {
      onDisconnect();
    }
  };

  if (isConnected && selectedProvider) {
    return (
      <div 
        data-testid="wallet-connected-status"
        className="flex items-center justify-between p-4 bg-gradient-to-r from-green-50 to-emerald-50 border border-green-200 rounded-xl"
      >
        <div className="flex items-center gap-3">
          <div className="p-2 bg-white rounded-lg shadow-sm">
            <Wallet className="h-5 w-5 text-emerald-600" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gray-700">Connected to</span>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </div>
            <span className="text-base font-semibold text-gray-900">{selectedProvider.info.name}</span>
          </div>
        </div>
        {onDisconnect && (
          <button
            data-testid="wallet-disconnect-button"
            onClick={handleDisconnect}
            className="px-4 py-2 text-sm font-medium text-red-600 bg-white border border-red-200 rounded-lg hover:bg-red-50 hover:border-red-300 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 transition-all duration-200 flex items-center gap-2"
            title="Disconnect wallet"
          >
            <LogOut className="h-4 w-4" />
            <span>Disconnect</span>
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="relative">
      <div className="relative">
        <div className="absolute left-4 top-1/2 transform -translate-y-1/2 pointer-events-none">
          <Wallet className="h-5 w-5 text-gray-400" />
        </div>
        <select
          data-testid="wallet-selector-dropdown"
          value={selectedProvider?.info.uuid || ""}
          onChange={handleChange}
          disabled={isConnecting || providers.length === 0}
          className={`
            w-full pl-12 pr-12 py-3.5 
            bg-white border rounded-xl
            text-base font-medium
            appearance-none cursor-pointer
            transition-all duration-200
            ${isConnecting 
              ? 'border-blue-300 bg-blue-50' 
              : 'border-gray-200 hover:border-gray-300 focus:border-blue-500 focus:ring-4 focus:ring-blue-100'
            }
            disabled:opacity-50 disabled:cursor-not-allowed disabled:bg-gray-50
            focus:outline-none
          `}
        >
          <option value="">
            {isConnecting
              ? "Connecting to wallet..."
              : providers.length > 0
              ? "Select a wallet"
              : "No wallets detected"}
          </option>
          {providers.map((provider, index) => (
            <option 
              key={provider.info.uuid} 
              value={provider.info.uuid}
              data-testid={`wallet-selector-option-${index}`}
            >
              {provider.info.name}
            </option>
          ))}
        </select>
        <div className="absolute right-4 top-1/2 transform -translate-y-1/2 pointer-events-none">
          {isConnecting ? (
            <Loader2 className="h-5 w-5 animate-spin text-blue-600" />
          ) : (
            <ChevronDown className="h-5 w-5 text-gray-400" />
          )}
        </div>
      </div>
      
      {providers.length === 0 && (
        <p 
          data-testid="wallet-no-wallets-message"
          className="mt-3 text-sm text-gray-500"
        >
          Please install a Web3 wallet extension like MetaMask to continue.
        </p>
      )}
    </div>
  );
}
