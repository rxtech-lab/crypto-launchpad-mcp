export interface EIP6963Provider {
  info: {
    uuid: string;
    name: string;
    icon: string;
    rdns: string;
  };
  provider: any; // EIP-1193 provider
}

export interface TransactionMetadata {
  key: string;
  value: string;
}

export interface TransactionDeployment {
  title: string;
  description: string;
  data: string;
  value: string;
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