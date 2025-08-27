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
  contractAddress?: string; // Added to track deployed contract address
  transactionHash?: string; // Added to track transaction hash
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
