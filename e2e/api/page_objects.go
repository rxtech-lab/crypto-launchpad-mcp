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
	// Wait for the main content div to be visible
	err := chromedp.Run(ctx,
		chromedp.WaitVisible("#content", chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("page did not load: %w", err)
	}

	// Wait for JavaScript to load session data and replace the loading state
	// The deploy-uniswap.js script will replace the content with deployment details
	err = chromedp.Run(ctx,
		chromedp.WaitVisible("h2", chromedp.ByQuery), // Wait for any h2 element (deployment details header)
	)
	if err != nil {
		return fmt.Errorf("deployment details did not load: %w", err)
	}

	return nil
}

// WaitForWalletSelection waits for the wallet selection dropdown to be ready
func (p *UniswapDeploymentPage) WaitForWalletSelection() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible("#wallet-select", chromedp.ByID),
	)
}

// SelectWallet selects a wallet from the dropdown
func (p *UniswapDeploymentPage) SelectWallet(walletUUID string) error {
	// Wait for wallet select to be available
	err := p.WaitForWalletSelection()
	if err != nil {
		return err
	}

	// Use JavaScript to set the value and trigger change event
	js := fmt.Sprintf(`
		const select = document.getElementById('wallet-select');
		select.value = '%s';
		select.dispatchEvent(new Event('change', { bubbles: true }));
	`, walletUUID)

	return chromedp.Run(p.ctx,
		chromedp.Evaluate(js, nil),
	)
}

// ClickConnectWallet clicks the connect wallet button
func (p *UniswapDeploymentPage) ClickConnectWallet() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible("#connect-button", chromedp.ByID),
		chromedp.Click("#connect-button", chromedp.ByID),
	)
}

// WaitForWalletConnection waits for the wallet to be connected
func (p *UniswapDeploymentPage) WaitForWalletConnection() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	// Wait for connection status to show success
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(".bg-green-50", chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("wallet connection failed: %w", err)
	}

	// Verify the sign button is visible
	err = chromedp.Run(ctx,
		chromedp.WaitVisible("#sign-button", chromedp.ByID),
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
		chromedp.WaitVisible("#sign-button", chromedp.ByID),
		chromedp.Click("#sign-button", chromedp.ByID),
	)
}

func (p *UniswapDeploymentPage) WaitForSuccessState() error {
	// wait for success-state
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible("#success-state", chromedp.ByID),
	)
}

// GetContractAddress gets a contract address from the success state
func (p *UniswapDeploymentPage) GetContractAddress(contractType string) (string, error) {
	selector := fmt.Sprintf("#%s-address", contractType)

	var text string
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByID),
		chromedp.Text(selector, &text, chromedp.ByID),
	)
	if err != nil {
		return "", fmt.Errorf("contract address not found for %s: %w", contractType, err)
	}

	if text == "Loading..." {
		return "", fmt.Errorf("contract address still loading for %s", contractType)
	}

	return text, nil
}

// GetAllContractAddresses gets all three contract addresses
func (p *UniswapDeploymentPage) GetAllContractAddresses() (map[string]string, error) {
	addresses := make(map[string]string)
	contracts := []string{"weth", "factory", "router"}

	for _, contract := range contracts {
		addr, err := p.GetContractAddress(contract)
		if err != nil {
			return nil, err
		}
		addresses[contract] = addr
	}

	return addresses, nil
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
