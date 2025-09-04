import { useQuery } from '@tanstack/react-query'
import { BrowserProvider, Contract, formatUnits } from 'ethers'
import type { EIP6963Provider } from '../types/wallet'

// ERC-20 ABI for balance, symbol, and decimals
const ERC20_ABI = [
  'function balanceOf(address owner) view returns (uint256)',
  'function symbol() view returns (string)',
  'function decimals() view returns (uint8)',
  'function name() view returns (string)',
]

interface TokenInfo {
  balance: string
  formattedBalance: string
  symbol: string
  decimals: number
  name: string
}

interface UseTokenBalanceOptions {
  contractAddress?: string
  walletAddress?: string
  provider?: EIP6963Provider
  chainId?: number
  enabled?: boolean
}

export function useTokenBalance({
  contractAddress,
  walletAddress,
  provider,
  chainId,
  enabled = true,
}: UseTokenBalanceOptions) {
  return useQuery<TokenInfo>({
    queryKey: ['tokenBalance', contractAddress, walletAddress, chainId],
    queryFn: async (): Promise<TokenInfo> => {
      if (!contractAddress || !walletAddress || !provider) {
        throw new Error('Missing required parameters')
      }

      const ethersProvider = new BrowserProvider(provider.provider)

      // Check if it's a contract address (has code) or if it's the zero address (native ETH)
      const isNativeETH = contractAddress === '0x0000000000000000000000000000000000000000' ||
                          contractAddress.toLowerCase() === '0x0'

      if (isNativeETH) {
        // Query native ETH balance
        const balance = await ethersProvider.getBalance(walletAddress)
        return {
          balance: balance.toString(),
          formattedBalance: formatUnits(balance, 18),
          symbol: 'ETH',
          decimals: 18,
          name: 'Ether',
        }
      }

      // Check if the address has contract code
      const code = await ethersProvider.getCode(contractAddress)
      if (code === '0x') {
        throw new Error('Address is not a contract')
      }

      // Query ERC-20 token
      const contract = new Contract(contractAddress, ERC20_ABI, ethersProvider)

      const [balance, symbol, decimals, name] = await Promise.all([
        contract.balanceOf(walletAddress),
        contract.symbol(),
        contract.decimals(),
        contract.name(),
      ])

      const formattedBalance = formatUnits(balance, decimals)

      return {
        balance: balance.toString(),
        formattedBalance,
        symbol,
        decimals,
        name,
      }
    },
    enabled: enabled && !!contractAddress && !!walletAddress && !!provider,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // Refetch every minute when component is focused
  })
}

// Helper function to format balance with proper decimals
export function formatTokenBalance(balance: string, decimals: number, symbol: string): string {
  try {
    const formatted = formatUnits(balance, decimals)
    const numValue = parseFloat(formatted)
    
    if (numValue === 0) return `0 ${symbol}`
    if (numValue < 0.000001) return `<0.000001 ${symbol}`
    
    // Format with appropriate decimal places
    const formattedValue = numValue.toLocaleString('en-US', {
      minimumFractionDigits: 0,
      maximumFractionDigits: decimals > 6 ? 6 : decimals,
    })
    
    return `${formattedValue} ${symbol}`
  } catch (error) {
    console.error('Error formatting token balance:', error)
    return `${balance} ${symbol}`
  }
}