import type { TransactionDeployment } from "../types/wallet";

export function formatAddress(address: string): string {
  if (!address || address.length < 10) return address;
  return `${address.substring(0, 6)}...${address.substring(
    address.length - 4
  )}`;
}

export function parseTransactionData(data: string): any {
  try {
    // If data is a JSON string, parse it
    if (data.startsWith("{")) {
      return JSON.parse(data);
    }
    // Otherwise return as-is (could be hex data)
    return data;
  } catch (error) {
    console.error("Failed to parse transaction data:", error);
    return data;
  }
}

export function prepareTransaction(deployment: TransactionDeployment) {
  return {
    to: undefined, // For contract deployment
    data: deployment.data,
    value: deployment.value,
  };
}

export function getChainName(chainId: number): string {
  const chains: Record<number, string> = {
    1: "Ethereum Mainnet",
    5: "Goerli Testnet",
    11155111: "Sepolia Testnet",
    137: "Polygon",
    80001: "Mumbai Testnet",
    10: "Optimism",
    420: "Optimism Goerli",
    42161: "Arbitrum One",
    421613: "Arbitrum Goerli",
    8453: "Base",
    84531: "Base Goerli",
  };

  return chains[chainId] || `Chain ${chainId}`;
}

export function formatEther(value: string): string {
  try {
    // Convert wei to ETH by dividing by 10^18
    const weiValue = BigInt(value);
    const ethValue = Number(weiValue) / Math.pow(10, 18);

    if (ethValue === 0) return "0";
    if (ethValue < 0.000001) return "<0.000001";
    return ethValue.toFixed(6).replace(/\.?0+$/, "");
  } catch {
    return value;
  }
}

export function isValidAddress(address: string): boolean {
  return /^0x[a-fA-F0-9]{40}$/.test(address);
}

export function truncateHash(hash: string, length: number = 10): string {
  if (!hash || hash.length <= length * 2) return hash;
  return `${hash.substring(0, length)}...${hash.substring(
    hash.length - length
  )}`;
}
