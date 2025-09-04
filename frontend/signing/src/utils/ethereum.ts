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

export function formatTokenBalance(value: string, decimals: number = 18): string {
  try {
    // Convert token units to decimal by dividing by 10^decimals
    const tokenValue = BigInt(value);
    const divisor = BigInt(10) ** BigInt(decimals);
    
    // Handle the division with proper decimal places
    const integerPart = tokenValue / divisor;
    const fractionalPart = tokenValue % divisor;
    
    if (fractionalPart === BigInt(0)) {
      return integerPart.toString();
    }
    
    // Convert fractional part to decimal string
    const fractionalStr = fractionalPart.toString().padStart(decimals, '0');
    const decimalValue = Number(integerPart) + Number(`0.${fractionalStr}`);
    
    // Format with appropriate precision
    if (decimalValue === 0) return "0";
    if (decimalValue < 0.000001) return "<0.000001";
    
    // Show more precision for smaller amounts, less for larger amounts
    let precision = 6;
    if (decimalValue >= 1000) precision = 2;
    else if (decimalValue >= 1) precision = 4;
    
    return decimalValue.toFixed(precision).replace(/\.?0+$/, "");
  } catch (error) {
    console.error("Error formatting token balance:", error);
    return value;
  }
}
