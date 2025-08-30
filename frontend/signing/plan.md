# Transaction Signing App - Development Plan

## Overview

Rebuild the transaction signing application using React with TypeScript, ethers.js for Ethereum interactions, and Lucide React for icons. The app will handle EIP-6963 wallet discovery, multi-transaction signing, and provide real-time status updates.

## Tech Stack

- **React 19** with TypeScript
- **ethers.js** - Ethereum interactions and wallet management
- **Lucide React** - Icon library for status indicators
- **Tailwind CSS** - Styling and animations
- **Vite** - Build tool

## Dependencies to Install

```bash
npm install ethers lucide-react
```

## Architecture

### File Structure

```
src/
├── types/
│   └── wallet.ts          # TypeScript interfaces and types
├── hooks/
│   ├── useWallet.ts       # Wallet management with ethers.js
│   └── useTransaction.ts  # Transaction queue and status management
├── components/
│   ├── WalletSelector.tsx     # Wallet discovery and selection
│   ├── ConnectionStatus.tsx   # Connection state display
│   ├── TransactionList.tsx    # List of transactions with status
│   ├── TransactionSigner.tsx  # Sign and send transactions
│   ├── MetadataDisplay.tsx    # Session metadata display
│   └── ErrorDisplay.tsx       # Error handling component
├── utils/
│   └── ethereum.ts        # Ethereum utility functions
├── App.tsx                # Main application component
├── index.css              # Tailwind directives and animations
└── main.tsx               # Application entry point
```

## Component Details

### 1. Types (`src/types/wallet.ts`)

```typescript
interface EIP6963Provider {
  info: {
    uuid: string;
    name: string;
    icon: string;
    rdns: string;
  };
  provider: any; // EIP-1193 provider
}

interface TransactionMetadata {
  key: string;
  value: string;
}

interface TransactionDeployment {
  title: string;
  description: string;
  data: string;
  value: string;
}

interface TransactionSession {
  id: string;
  metadata: TransactionMetadata[];
  status: "pending" | "confirmed" | "failed";
  chain_type: "ethereum" | "solana";
  transaction_deployments: TransactionDeployment[];
  chain_id: number;
  created_at: string;
  expires_at: string;
}

type TransactionStatus = "waiting" | "pending" | "confirmed" | "failed";
```

### 2. Wallet Hook (`src/hooks/useWallet.ts`)

**Functionality:**

- EIP-6963 wallet discovery via event listeners
- Connect wallet using ethers.js BrowserProvider
- Network switching support
- Transaction signing with ethers Signer
- Wallet state management (connected, account, chainId)

**Key Methods:**

- `discoverWallets()` - Listen for EIP-6963 announcements
- `connectWallet(uuid)` - Connect to selected wallet
- `signTransaction(tx)` - Sign and send transaction
- `switchNetwork(chainId)` - Switch to required network

### 3. Transaction Hook (`src/hooks/useTransaction.ts`)

**Functionality:**

- Load session data from meta tags or API
- Manage transaction queue
- Track individual transaction status
- Execute transactions sequentially
- Update UI after each transaction

**State Management:**

```typescript
{
  session: TransactionSession | null;
  transactionStatuses: Map<number, TransactionStatus>;
  currentIndex: number;
  error: Error | null;
}
```

### 4. Components

#### WalletSelector (`src/components/WalletSelector.tsx`)

- Dropdown showing discovered wallets
- Connect button with `Wallet` icon
- Loading state with `Loader2` spinning icon
- Disabled state when already connected

#### ConnectionStatus (`src/components/ConnectionStatus.tsx`)

**Connected State:**

- Green card with `CheckCircle2` icon
- Display wallet address (formatted)
- Show chain ID with `Link` icon

**Disconnected State:**

- Amber card with `AlertTriangle` icon
- Warning message

#### TransactionList (`src/components/TransactionList.tsx`)

**Features:**

- Display all transactions from session
- Status icons per transaction:
  - `CheckCircle2` (green) - Completed
  - `Loader2` (animated spin) - In progress
  - `XCircle` (red) - Failed
  - `Clock` (gray) - Waiting
- Transaction details (title, description, value)
- Progress indicators with animations

**Layout:**

```jsx
<div className="space-y-3">
  {transactions.map((tx, index) => (
    <div className="flex items-center p-4 rounded-lg border">
      <StatusIcon status={statuses.get(index)} />
      <div className="flex-grow ml-4">
        <h4>{tx.title}</h4>
        <p className="text-sm text-gray-500">{tx.description}</p>
      </div>
      <span className="font-mono text-sm">{tx.value} ETH</span>
    </div>
  ))}
</div>
```

#### TransactionSigner (`src/components/TransactionSigner.tsx`)

**States:**

- Idle: "Sign & Send Transactions" with `Send` icon
- Signing: "Signing X of Y..." with `Loader2` spinner
- Success: "All Transactions Complete" with `CheckCircle2`
- Error: Error message with `AlertCircle`

**Features:**

- Progress counter during signing
- Disable during execution
- Error handling with retry option

#### MetadataDisplay (`src/components/MetadataDisplay.tsx`)

- Header with `Info` icon
- Grid layout for metadata key-value pairs
- Clean card design
- `FileText` icon for document-related metadata

#### ErrorDisplay (`src/components/ErrorDisplay.tsx`)

- Red alert box with `XCircle` icon
- Error message display
- Retry button with `RefreshCw` icon
- Expandable details with `ChevronDown/Up`

### 5. Main App (`src/App.tsx`)

**Structure:**

```jsx
<div className="min-h-screen bg-gray-50 p-6">
  <div className="max-w-4xl mx-auto space-y-6">
    <Header />
    <WalletSelector />
    <ConnectionStatus />
    <MetadataDisplay />
    <TransactionList />
    <TransactionSigner />
  </div>
</div>
```

### 6. Utilities (`src/utils/ethereum.ts`)

- `formatAddress(address)` - Format as 0x1234...5678
- `parseTransactionData(data)` - Parse transaction data
- `prepareTransaction(deployment)` - Prepare tx for ethers
- `getChainName(chainId)` - Get readable chain name

### 7. Styling (`src/index.css`)

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Custom animations */
@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes pulse-soft {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

@keyframes slide-up {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.animate-fade-in {
  animation: fade-in 0.3s ease-out;
}

.animate-pulse-soft {
  animation: pulse-soft 2s ease-in-out infinite;
}

.animate-slide-up {
  animation: slide-up 0.4s ease-out;
}
```

## Icon Usage Guide

### Status Icons

- `CheckCircle2` - Success/Completed (text-green-500)
- `XCircle` - Failed/Error (text-red-500)
- `Loader2` - Loading/Pending (animate-spin text-blue-500)
- `Clock` - Waiting/Queued (text-gray-400)
- `AlertTriangle` - Warning (text-amber-500)
- `AlertCircle` - Error state (text-red-500)

### Action Icons

- `Send` - Send transaction
- `RefreshCw` - Retry action
- `Wallet` - Wallet selector
- `Link` - Network/Chain indicator

### Info Icons

- `Info` - Information sections
- `FileText` - Document/Contract details
- `Layers` - Multiple transactions
- `Activity` - Progress indicator
- `ChevronDown/ChevronUp` - Expand/Collapse

## Transaction Flow

1. **Page Load**

   - Read session data from meta tag
   - Initialize EIP-6963 wallet discovery
   - Display session metadata and transaction list

2. **Wallet Connection**

   - User selects wallet from dropdown
   - Connect using ethers.js BrowserProvider
   - Display connection status

3. **Transaction Signing**

   - User clicks "Sign & Send Transactions"
   - Process each transaction sequentially:
     - Update status to 'pending'
     - Sign with ethers Signer
     - Send transaction
     - Wait for confirmation
     - Update status to 'confirmed' or 'failed'
   - Show progress (X of Y transactions)

4. **Completion**
   - Display all transaction results
   - Show success message or error details
   - Provide retry option if needed

## Error Handling

- **Wallet Errors:**

  - No wallet installed
  - User rejection
  - Network mismatch

- **Transaction Errors:**

  - Insufficient gas
  - Transaction revert
  - Network issues

- **Session Errors:**
  - Invalid session
  - Expired session
  - Missing data

## Testing Considerations

- Test with multiple wallets (MetaMask, Rainbow, etc.)
- Test transaction rejection scenarios
- Test network switching
- Test multiple transaction sequences
- Test error recovery
- Test session expiration

## Performance Optimizations

- Lazy load wallet icons
- Debounce wallet discovery events
- Memoize expensive computations
- Use React.memo for pure components
- Optimize re-renders with proper dependencies

## Accessibility

- Proper ARIA labels for icons
- Keyboard navigation support
- Focus management
- Screen reader announcements
- Color contrast compliance

## Future Enhancements

- Transaction simulation before signing
- Gas estimation and optimization
- Batch transaction support
- Transaction history
- Export transaction receipts
- Mobile responsive improvements
