export interface EIP6963Provider {
  info: {
    uuid: string;
    name: string;
    icon: string;
    rdns: string;
  };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  provider: any; // EIP-1193 provider
}

export interface TransactionMetadata {
  key: string;
  value: string;
  title: string;
  description: string;
}

export interface TransactionDeployment {
  title: string;
  description: string;
  data: string;
  value: string;
  receiver?: string; // Added to track receiver address
  transactionHash?: string; // Added to track transaction hash
  contractCode?: string; // Added to track contract code
  contractAddress?: string; // Added to track contract address
  rawContractArguments?: string; // Added to track raw contract arguments
  showBalanceBeforeDeployment?: boolean; // Added to track if balance should be shown before deployment
  showBalanceAfterDeployment?: boolean; // Added to track if balance should be shown after deployment
  transactionType:
    | "regular"
    | "token_swap"
    | "add_liquidity"
    | "remove_liquidity"; // Added to track transaction type
}

export interface BlockchainNetwork {
  rpc: string;
  chain_id: number;
  name: string;
  type: "ethereum" | "solana";
}

export interface TransactionSession {
  id: string;
  metadata: TransactionMetadata[];
  status: "pending" | "confirmed" | "failed";
  chain_type: "ethereum" | "solana";
  transaction_deployments: TransactionDeployment[];
  chain_id: number;
  created_at: string;
  expires_at: string;
  // balances where key is the contract address and value is the balance
  // if the balance is null, it means that we need to fetch the balance from the blockchain
  balances: Record<string, string | null>;
}

export type TransactionStatus = "waiting" | "pending" | "confirmed" | "failed";

export interface WalletState {
  providers: EIP6963Provider[];
  selectedProvider: EIP6963Provider | null;
  account: string | null;
  chainId: number | null;
  isConnected: boolean;
  isConnecting: boolean;
  error: Error | null;
}

export interface TransactionState {
  session: TransactionSession | null;
  transactionStatuses: Map<number, TransactionStatus>;
  currentIndex: number;
  error: Error | null;
  isExecuting: boolean;
}
