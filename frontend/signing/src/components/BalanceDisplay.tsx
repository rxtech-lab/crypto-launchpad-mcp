import { Activity } from "lucide-react";
import { formatTokenBalance } from "../utils/ethereum";

interface BalanceDisplayProps {
  balances: Record<string, string | null>;
  chainId?: number | null;
  onRefresh?: () => void;
}

interface TokenInfo {
  name: string;
  symbol: string;
  decimals: number;
}

export function BalanceDisplay({ balances, chainId, onRefresh }: BalanceDisplayProps) {
  // Token metadata for common tokens
  const getTokenInfo = (address: string): TokenInfo => {
    // Native token (empty string or "0x0")
    if (!address || address === "0x0" || address.toLowerCase() === "0x0000000000000000000000000000000000000000") {
      return { name: "Ethereum", symbol: "ETH", decimals: 18 };
    }

    // Common token mappings (can be extended)
    const tokenMap: Record<string, TokenInfo> = {
      // USDC on various chains
      "0xA0b86a33E6441000000000000000000000000000": { name: "USD Coin", symbol: "USDC", decimals: 6 },
      "0xa0b86a33e6441000000000000000000000000000": { name: "USD Coin", symbol: "USDC", decimals: 6 },
      // USDT 
      "0xdAC17F958D2ee523a2206206994597C13D831ec7": { name: "Tether USD", symbol: "USDT", decimals: 6 },
      "0xdac17f958d2ee523a2206206994597c13d831ec7": { name: "Tether USD", symbol: "USDT", decimals: 6 },
      // DAI
      "0x6B175474E89094C44Da98b954EedeAC495271d0F": { name: "Dai Stablecoin", symbol: "DAI", decimals: 18 },
      "0x6b175474e89094c44da98b954eedeac495271d0f": { name: "Dai Stablecoin", symbol: "DAI", decimals: 18 },
      // WETH
      "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2": { name: "Wrapped Ether", symbol: "WETH", decimals: 18 },
      "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": { name: "Wrapped Ether", symbol: "WETH", decimals: 18 },
    };

    return tokenMap[address] || { name: "Token", symbol: "TOKEN", decimals: 18 };
  };

  // Filter out empty balances and format entries
  const balanceEntries = Object.entries(balances).filter(([_, balance]) => balance !== null);

  if (balanceEntries.length === 0) {
    return null;
  }

  return (
    <div className="px-6 py-4 border-b border-gray-200 bg-white">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-2">
          <Activity className="w-4 h-4 text-gray-600" />
          <span className="text-sm font-medium text-gray-700">Token Balances</span>
          {chainId && (
            <span className="text-xs text-gray-500">
              (Chain {chainId})
            </span>
          )}
        </div>
        
        {onRefresh && (
          <button
            onClick={onRefresh}
            className="flex items-center space-x-1 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded-md transition-colors"
            title="Refresh balances"
          >
            <Activity className="w-3 h-3" />
            <span>Refresh</span>
          </button>
        )}
      </div>

      <div className="space-y-2">
        {balanceEntries.map(([address, balance]) => {
          const tokenInfo = getTokenInfo(address);
          
          if (balance === null) {
            return (
              <div key={address} className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div className="w-2 h-2 bg-gray-300 rounded-full animate-pulse"></div>
                  <span className="text-sm text-gray-700">{tokenInfo.name}</span>
                </div>
                <div className="flex items-center space-x-1">
                  <div className="w-12 h-4 bg-gray-200 rounded animate-pulse"></div>
                  <span className="text-sm text-gray-500">{tokenInfo.symbol}</span>
                </div>
              </div>
            );
          }

          const formattedBalance = formatTokenBalance(balance, tokenInfo.decimals);
          
          return (
            <div key={address} className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-sm text-gray-700">{tokenInfo.name}</span>
              </div>
              <div className="flex items-center space-x-1">
                <span className="text-sm font-medium text-gray-900">
                  {formattedBalance}
                </span>
                <span className="text-sm text-gray-600">{tokenInfo.symbol}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}