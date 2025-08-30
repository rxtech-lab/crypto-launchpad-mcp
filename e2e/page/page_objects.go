package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// UniswapDeploymentPage represents the Uniswap deployment page
type UniswapDeploymentPage struct {
	ctx context.Context
}

// NewUniswapDeploymentPage creates a new page object
func NewUniswapDeploymentPage(ctx context.Context) *UniswapDeploymentPage {
	return &UniswapDeploymentPage{ctx: ctx}
}

// NavigateToSession navigates to the deployment session URL
func (p *UniswapDeploymentPage) NavigateToSession(baseURL, sessionID string) error {
	url := fmt.Sprintf("%s/deploy-uniswap/%s", baseURL, sessionID)
	return chromedp.Run(p.ctx, chromedp.Navigate(url))
}

// WaitForPageLoad waits for the page to load and initial content to appear
func (p *UniswapDeploymentPage) WaitForPageLoad() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	// Wait for the metadata container to be visible (indicates page loaded)
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="metadata-container"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("page did not load: %w", err)
	}

	// Wait for transaction list to be visible
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="transaction-list-container"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("transaction list did not load: %w", err)
	}

	return nil
}

// WaitForWalletSelection waits for the wallet selection dropdown to be ready
func (p *UniswapDeploymentPage) WaitForWalletSelection() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="wallet-selector-dropdown"]`, chromedp.ByQuery),
	)
}

// SelectWallet selects a wallet from the dropdown by option value (provider UUID)
func (p *UniswapDeploymentPage) SelectWallet(walletValue string) error {
	// Wait for wallet select to be available
	err := p.WaitForWalletSelection()
	if err != nil {
		return fmt.Errorf("failed to wait for wallet selection: %w", err)
	}

	// 10 seconds timeout
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	// Select the wallet option by setting the select element's value
	err = chromedp.Run(ctx,
		chromedp.SetValue(`[data-testid="wallet-selector-dropdown"]`, walletValue, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to select wallet with value '%s': %w", walletValue, err)
	}

	return nil
}

// ClickConnectWallet clicks the connect wallet button (wallet connection now automatic after selection)
func (p *UniswapDeploymentPage) ClickConnectWallet() error {
	// In the new frontend, wallet connection happens automatically after selection
	// Just wait for the wallet to be connected
	return p.WaitForWalletConnection()
}

// WaitForWalletConnection waits for the wallet to be connected
func (p *UniswapDeploymentPage) WaitForWalletConnection() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	// Wait for connection status to show success
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="wallet-connected-status"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("wallet connection failed: %w", err)
	}

	// Verify the sign button is visible
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="transaction-sign-button"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("sign button not visible after connection: %w", err)
	}

	return nil
}

// ClickDeployButton clicks the main deploy button
func (p *UniswapDeploymentPage) ClickDeployButton() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="transaction-sign-button"]`, chromedp.ByQuery),
		chromedp.Click(`[data-testid="transaction-sign-button"]`, chromedp.ByQuery),
	)
}

func (p *UniswapDeploymentPage) WaitForSuccessState() error {
	// wait for success-state
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`[data-testid="transaction-success-message"]`, chromedp.ByQuery),
	)
}

// GetContractAddress gets a contract address from the success state
func (p *UniswapDeploymentPage) GetContractAddress(contractIndex int) (string, error) {
	selector := fmt.Sprintf(`[data-testid="deployed-contract-address-%d"]`, contractIndex)

	var text string
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Text(selector, &text, chromedp.ByQuery),
	)
	if err != nil {
		return "", fmt.Errorf("contract address not found for index %d: %w", contractIndex, err)
	}

	if text == "Loading..." || text == "" {
		return "", fmt.Errorf("contract address still loading for index %d", contractIndex)
	}

	return text, nil
}

// WaitForErrorState waits for an error to be displayed
func (p *UniswapDeploymentPage) WaitForErrorState() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="error-display-container"]`, chromedp.ByQuery),
	)
}

// GetErrorMessage gets the error message text
func (p *UniswapDeploymentPage) GetErrorMessage() (string, error) {
	var text string
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="error-message"]`, chromedp.ByQuery),
		chromedp.Text(`[data-testid="error-message"]`, &text, chromedp.ByQuery),
	)
	return text, err
}

// ClickRetryButton clicks the retry button if visible
func (p *UniswapDeploymentPage) ClickRetryButton() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`[data-testid="error-retry-button"]`, chromedp.ByQuery),
		chromedp.Click(`[data-testid="error-retry-button"]`, chromedp.ByQuery),
	)
}

// WaitForNoWalletsMessage waits for the no wallets detected message
func (p *UniswapDeploymentPage) WaitForNoWalletsMessage() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`[data-testid="wallet-no-wallets-message"]`, chromedp.ByQuery),
	)
}

// CheckConnectionStatus checks if wallet is connected or shows warning
func (p *UniswapDeploymentPage) CheckConnectionStatus() (string, error) {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	// Check if wallet is connected
	var connectedVisible bool
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[data-testid="wallet-connected-info"]') !== null`, &connectedVisible),
	)
	if err != nil {
		return "error", err
	}

	if connectedVisible {
		return "connected", nil
	}

	// Check if showing not connected warning
	var warningVisible bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[data-testid="wallet-not-connected-warning"]') !== null`, &warningVisible),
	)
	if err != nil {
		return "error", err
	}

	if warningVisible {
		return "not_connected", nil
	}

	return "unknown", nil
}

// GetTransactionStatus gets the status of a specific transaction
func (p *UniswapDeploymentPage) GetTransactionStatus(transactionIndex int) (string, error) {
	// Check if transaction has success status
	successSelector := fmt.Sprintf(`[data-testid="transaction-status-icon-%d"]`, transactionIndex)

	var text string
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(successSelector, chromedp.ByQuery),
		chromedp.Text(successSelector, &text, chromedp.ByQuery),
	)
	if err != nil {
		return "unknown", err
	}

	return text, nil
}

// TakeScreenshot takes a screenshot of the current page state
func (p *UniswapDeploymentPage) TakeScreenshot(filename string) error {
	var buf []byte
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return err
	}

	// Save screenshot to file
	return os.WriteFile(filename, buf, 0644)
}

// GetPageTitle gets the page title
func (p *UniswapDeploymentPage) GetPageTitle() (string, error) {
	var title string
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.Title(&title),
	)
	return title, err
}

// WaitWithTimeout waits for a condition with a custom timeout
func (p *UniswapDeploymentPage) WaitWithTimeout(timeout time.Duration, tasks ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	return chromedp.Run(ctx, tasks...)
}

// GetPageHTML returns the current HTML content of the page
func (p *UniswapDeploymentPage) GetPageHTML() (string, error) {
	var html string
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	return html, err
}

// LogPageState logs the current page state for debugging
func (p *UniswapDeploymentPage) LogPageState(testName string) error {
	html, err := p.GetPageHTML()
	if err != nil {
		return fmt.Errorf("failed to get page HTML: %w", err)
	}

	// Write HTML to file for debugging
	filename := fmt.Sprintf("debug_%s.html", testName)
	err = os.WriteFile(filename, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	fmt.Printf("DEBUG: Page HTML saved to %s\n", filename)
	return nil
}
