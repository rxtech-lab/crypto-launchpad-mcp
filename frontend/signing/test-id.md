# Test IDs Reference for E2E Testing

This document provides a comprehensive reference of all `data-testid` attributes added to the React components in the signing frontend for E2E testing purposes.

## WalletSelector Component

### Purpose

Handles wallet selection, connection status, and disconnect functionality.

### Test IDs

| Test ID                          | Element                   | Purpose                                |
| -------------------------------- | ------------------------- | -------------------------------------- |
| `wallet-selector-dropdown`       | Main wallet dropdown      | Select wallet from available providers |
| `wallet-selector-option-{index}` | Individual wallet options | Select specific wallet (index-based)   |
| `wallet-connected-status`        | Connected wallet display  | Verify wallet connection state         |
| `wallet-disconnect-button`       | Disconnect button         | Disconnect from current wallet         |
| `wallet-no-wallets-message`      | No wallets message        | Verify no wallets detected scenario    |

### Example Usage

```javascript
// Select MetaMask (assuming it's the first option)
await page.click('[data-testid="wallet-selector-dropdown"]');
await page.click('[data-testid="wallet-selector-option-0"]');

// Verify wallet is connected
await expect(
  page.locator('[data-testid="wallet-connected-status"]')
).toBeVisible();

// Disconnect wallet
await page.click('[data-testid="wallet-disconnect-button"]');
```

## TransactionSigner Component

### Purpose

Main transaction signing interface with error handling and status display.

### Test IDs

| Test ID                       | Element                    | Purpose                        |
| ----------------------------- | -------------------------- | ------------------------------ |
| `transaction-sign-button`     | Main sign & send button    | Initiate transaction signing   |
| `transaction-error-message`   | Error message container    | Verify error state display     |
| `transaction-error-details`   | Error details text         | Access specific error messages |
| `transaction-retry-button`    | Retry button               | Retry failed transactions      |
| `transaction-status-message`  | Status messages            | Verify various status states   |
| `transaction-success-message` | Success completion message | Confirm successful completion  |

### Example Usage

```javascript
// Sign transactions
await page.click('[data-testid="transaction-sign-button"]');

// Handle errors
if (
  await page.locator('[data-testid="transaction-error-message"]').isVisible()
) {
  const errorText = await page
    .locator('[data-testid="transaction-error-details"]')
    .textContent();
  console.log("Error:", errorText);
  await page.click('[data-testid="transaction-retry-button"]');
}

// Verify success
await expect(
  page.locator('[data-testid="transaction-success-message"]')
).toBeVisible();
```

## HorizontalStepper Component

### Purpose

Visual progress indicator for multi-step processes.

### Test IDs

| Test ID                          | Element                              | Purpose                         |
| -------------------------------- | ------------------------------------ | ------------------------------- |
| `stepper-container`              | Main stepper container               | Access entire stepper component |
| `stepper-step-{index}`           | Individual step elements             | Target specific steps           |
| `stepper-step-indicator-{index}` | Step indicators (numbers/checkmarks) | Verify step completion status   |
| `stepper-step-label-{index}`     | Step labels                          | Read step descriptions          |
| `stepper-mobile-label`           | Mobile step label display            | Mobile-specific step info       |

### Example Usage

```javascript
// Verify current step
await expect(
  page.locator('[data-testid="stepper-step-indicator-0"]')
).toHaveClass(/bg-blue-600/);
await expect(
  page.locator('[data-testid="stepper-step-indicator-1"]')
).toHaveClass(/bg-green-500/);

// Check step labels
const stepLabel = await page
  .locator('[data-testid="stepper-step-label-0"]')
  .textContent();
expect(stepLabel).toBe("Connect Wallet");
```

## ErrorDisplay Component

### Purpose

Reusable error display with expandable details and retry functionality.

### Test IDs

| Test ID                   | Element                  | Purpose                           |
| ------------------------- | ------------------------ | --------------------------------- |
| `error-display-container` | Main error container     | Verify error component visibility |
| `error-message`           | Error message text       | Access error message content      |
| `error-details-toggle`    | Show/hide details button | Toggle error details visibility   |
| `error-stack-trace`       | Stack trace display      | Access detailed error information |
| `error-retry-button`      | Retry button             | Retry failed operations           |

### Example Usage

```javascript
// Handle error display
if (await page.locator('[data-testid="error-display-container"]').isVisible()) {
  const errorMessage = await page
    .locator('[data-testid="error-message"]')
    .textContent();

  // Expand details if available
  if (await page.locator('[data-testid="error-details-toggle"]').isVisible()) {
    await page.click('[data-testid="error-details-toggle"]');
    const stackTrace = await page
      .locator('[data-testid="error-stack-trace"]')
      .textContent();
  }

  // Retry operation
  await page.click('[data-testid="error-retry-button"]');
}
```

## ConnectionStatus Component

### Purpose

Display wallet connection status and network information.

### Test IDs

| Test ID                        | Element                | Purpose                         |
| ------------------------------ | ---------------------- | ------------------------------- |
| `connection-status-container`  | Main status container  | Access entire connection status |
| `wallet-not-connected-warning` | Not connected warning  | Verify disconnected state       |
| `wallet-connected-info`        | Connected wallet info  | Verify connected state          |
| `wallet-address`               | Wallet address display | Access displayed wallet address |
| `chain-info`                   | Chain information      | Verify network/chain details    |
| `wrong-network-warning`        | Wrong network warning  | Verify network mismatch state   |

### Example Usage

```javascript
// Verify wallet connection
await expect(
  page.locator('[data-testid="wallet-connected-info"]')
).toBeVisible();

// Check wallet address
const address = await page
  .locator('[data-testid="wallet-address"]')
  .textContent();
expect(address).toMatch(/0x[a-fA-F0-9]{40}/); // Verify address format

// Check network
const chainInfo = await page
  .locator('[data-testid="chain-info"]')
  .textContent();
expect(chainInfo).toBe("Ethereum Mainnet");

// Handle wrong network
if (await page.locator('[data-testid="wrong-network-warning"]').isVisible()) {
  // Switch network logic
}
```

## TransactionList Component

### Purpose

Display list of transactions with status tracking and contract addresses.

### Test IDs

| Test ID                             | Element                      | Purpose                            |
| ----------------------------------- | ---------------------------- | ---------------------------------- |
| `transaction-list-container`        | Main container               | Access entire transaction list     |
| `transaction-item-{index}`          | Individual transaction items | Target specific transactions       |
| `transaction-status-icon-{index}`   | Status icons                 | Verify transaction status visually |
| `transaction-title-{index}`         | Transaction titles           | Access transaction descriptions    |
| `transaction-value-{index}`         | Transaction values           | Verify ETH amounts                 |
| `deployed-contract-address-{index}` | Contract addresses           | Access deployed contract addresses |
| `copy-address-button-{index}`       | Copy address buttons         | Copy contract addresses            |

### Example Usage

```javascript
// Verify transaction list
await expect(
  page.locator('[data-testid="transaction-list-container"]')
).toBeVisible();

// Check first transaction
await expect(page.locator('[data-testid="transaction-item-0"]')).toBeVisible();
const title = await page
  .locator('[data-testid="transaction-title-0"]')
  .textContent();
const value = await page
  .locator('[data-testid="transaction-value-0"]')
  .textContent();

// Wait for transaction confirmation and get contract address
await page.waitForSelector('[data-testid="deployed-contract-address-0"]');
const contractAddress = await page
  .locator('[data-testid="deployed-contract-address-0"]')
  .textContent();

// Copy contract address
await page.click('[data-testid="copy-address-button-0"]');
```

## MetadataDisplay Component

### Purpose

Display session information and metadata about the current transaction session.

### Test IDs

| Test ID                  | Element                   | Purpose                          |
| ------------------------ | ------------------------- | -------------------------------- |
| `metadata-container`     | Main container            | Access entire metadata display   |
| `session-id`             | Session ID display        | Verify session identifier        |
| `metadata-item-{index}`  | Individual metadata items | Target specific metadata entries |
| `metadata-title-{index}` | Metadata titles           | Access metadata field names      |
| `metadata-value-{index}` | Metadata values           | Access metadata field values     |

### Example Usage

```javascript
// Verify metadata display
await expect(page.locator('[data-testid="metadata-container"]')).toBeVisible();

// Get session ID
const sessionId = await page
  .locator('[data-testid="session-id"]')
  .textContent();
expect(sessionId).toMatch(/Session ID: [a-f0-9-]{36}/);

// Check metadata items
const metadataCount = await page
  .locator('[data-testid^="metadata-item-"]')
  .count();
for (let i = 0; i < metadataCount; i++) {
  const title = await page
    .locator(`[data-testid="metadata-title-${i}"]`)
    .textContent();
  const value = await page
    .locator(`[data-testid="metadata-value-${i}"]`)
    .textContent();
  console.log(`${title}: ${value}`);
}
```

## Best Practices

### 1. Index-Based IDs

- Use index-based test IDs for dynamic lists (`{index}` instead of `{uuid}`)
- Ensures predictable test selectors regardless of data
- Example: `transaction-item-0`, `stepper-step-1`

### 2. Hierarchical Naming

- Use component-specific prefixes for clarity
- Example: `wallet-selector-dropdown`, `transaction-sign-button`
- Avoid generic names like `button` or `container`

### 3. State-Specific IDs

- Different test IDs for different component states when needed
- Example: `wallet-connected-status` vs `wallet-not-connected-warning`

### 4. Consistent Patterns

- Follow consistent naming patterns across components
- Use kebab-case for all test IDs
- Include component context in the ID name

### 5. Meaningful Descriptions

- Test IDs should clearly indicate the element's purpose
- Examples:
  - `transaction-retry-button` (action-specific)
  - `error-message` (content-specific)
  - `stepper-step-indicator-0` (position and type specific)

### 6. Avoid Implementation Details

- Don't tie test IDs to CSS classes or styling
- Focus on functional purpose rather than visual appearance
- Test IDs should remain stable across UI changes

## Testing Scenarios

### Complete Wallet Connection Flow

```javascript
// 1. Select wallet
await page.click('[data-testid="wallet-selector-dropdown"]');
await page.click('[data-testid="wallet-selector-option-0"]');

// 2. Verify connection
await expect(
  page.locator('[data-testid="wallet-connected-status"]')
).toBeVisible();
await expect(
  page.locator('[data-testid="wallet-connected-info"]')
).toBeVisible();

// 3. Check address and network
const address = await page
  .locator('[data-testid="wallet-address"]')
  .textContent();
const network = await page.locator('[data-testid="chain-info"]').textContent();
```

### Transaction Signing Flow

```javascript
// 1. Verify transactions loaded
await expect(
  page.locator('[data-testid="transaction-list-container"]')
).toBeVisible();

// 2. Sign transactions
await page.click('[data-testid="transaction-sign-button"]');

// 3. Handle potential errors
await page.waitForSelector(
  '[data-testid="transaction-success-message"], [data-testid="transaction-error-message"]'
);

// 4. Verify completion
if (
  await page.locator('[data-testid="transaction-success-message"]').isVisible()
) {
  await expect(
    page.locator('[data-testid="deployed-contract-address-0"]')
  ).toBeVisible();
}
```

### Error Handling Flow

```javascript
// 1. Trigger error condition
// ... error-triggering action ...

// 2. Verify error display
await expect(
  page.locator('[data-testid="error-display-container"]')
).toBeVisible();

// 3. Get error details
const errorMessage = await page
  .locator('[data-testid="error-message"]')
  .textContent();

// 4. Expand details if needed
if (await page.locator('[data-testid="error-details-toggle"]').isVisible()) {
  await page.click('[data-testid="error-details-toggle"]');
  const stackTrace = await page
    .locator('[data-testid="error-stack-trace"]')
    .textContent();
}

// 5. Retry operation
await page.click('[data-testid="error-retry-button"]');
```

---

This documentation serves as a comprehensive reference for writing reliable E2E tests for the signing frontend components. All test IDs are designed to be stable, meaningful, and maintainable across component updates and refactoring.
