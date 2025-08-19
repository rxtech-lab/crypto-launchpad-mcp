package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/chromedp/chromedp"
)

// LiquidityPoolPage represents the page object for liquidity pool operations
type LiquidityPoolPage struct {
	ctx context.Context
}

// NewLiquidityPoolPage creates a new liquidity pool page object
func NewLiquidityPoolPage(ctx context.Context) *LiquidityPoolPage {
	return &LiquidityPoolPage{ctx: ctx}
}

// NavigateToCreatePoolSession navigates to the create pool session page
func (p *LiquidityPoolPage) NavigateToCreatePoolSession(baseURL, sessionID string) error {
	url := fmt.Sprintf("%s/pool/create/%s", baseURL, sessionID)
	return chromedp.Run(p.ctx,
		chromedp.Navigate(url),
	)
}

// NavigateToAddLiquiditySession navigates to the add liquidity session page
func (p *LiquidityPoolPage) NavigateToAddLiquiditySession(baseURL, sessionID string) error {
	url := fmt.Sprintf("%s/liquidity/add/%s", baseURL, sessionID)
	return chromedp.Run(p.ctx,
		chromedp.Navigate(url),
	)
}

// NavigateToRemoveLiquiditySession navigates to the remove liquidity session page
func (p *LiquidityPoolPage) NavigateToRemoveLiquiditySession(baseURL, sessionID string) error {
	url := fmt.Sprintf("%s/liquidity/remove/%s", baseURL, sessionID)
	return chromedp.Run(p.ctx,
		chromedp.Navigate(url),
	)
}

// NavigateToSwapSession navigates to the swap session page
func (p *LiquidityPoolPage) NavigateToSwapSession(baseURL, sessionID string) error {
	url := fmt.Sprintf("%s/swap/%s", baseURL, sessionID)
	return chromedp.Run(p.ctx,
		chromedp.Navigate(url),
	)
}

// WaitForPageLoad waits for the page to fully load
func (p *LiquidityPoolPage) WaitForPageLoad() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`#session-data`, chromedp.ByID),
	)
}

// GetPageTitle gets the page title
func (p *LiquidityPoolPage) GetPageTitle() (string, error) {
	var title string
	err := chromedp.Run(p.ctx,
		chromedp.Title(&title),
	)
	return title, err
}

// WaitForWalletSelection waits for wallet selection to be available
func (p *LiquidityPoolPage) WaitForWalletSelection() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`#wallet-select`, chromedp.ByID),
	)
}

// SelectWallet selects a wallet from the dropdown
func (p *LiquidityPoolPage) SelectWallet(walletName string) error {
	return chromedp.Run(p.ctx,
		chromedp.SetValue(`#wallet-select`, walletName, chromedp.ByID),
		chromedp.Sleep(500*time.Millisecond),
	)
}

// ClickConnectWallet clicks the connect wallet button
func (p *LiquidityPoolPage) ClickConnectWallet() error {
	return chromedp.Run(p.ctx,
		chromedp.Click(`#connect-wallet`, chromedp.ByID),
		chromedp.Sleep(1*time.Second),
	)
}

// WaitForWalletConnection waits for the wallet to be connected
func (p *LiquidityPoolPage) WaitForWalletConnection() error {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()
	return chromedp.Run(ctx,
		chromedp.WaitVisible(`#wallet-info`, chromedp.ByID),
	)
}

// ClickCreatePoolButton clicks the create pool button
func (p *LiquidityPoolPage) ClickCreatePoolButton() error {
	return chromedp.Run(p.ctx,
		chromedp.Click(`#create-pool-btn`, chromedp.ByID),
		chromedp.Sleep(1*time.Second),
	)
}

// ClickAddLiquidityButton clicks the add liquidity button
func (p *LiquidityPoolPage) ClickAddLiquidityButton() error {
	return chromedp.Run(p.ctx,
		chromedp.Click(`#add-liquidity-btn`, chromedp.ByID),
		chromedp.Sleep(1*time.Second),
	)
}

// ClickRemoveLiquidityButton clicks the remove liquidity button
func (p *LiquidityPoolPage) ClickRemoveLiquidityButton() error {
	return chromedp.Run(p.ctx,
		chromedp.Click(`#remove-liquidity-btn`, chromedp.ByID),
		chromedp.Sleep(1*time.Second),
	)
}

// ClickSwapButton clicks the swap button
func (p *LiquidityPoolPage) ClickSwapButton() error {
	return chromedp.Run(p.ctx,
		chromedp.Click(`#swap-btn`, chromedp.ByID),
		chromedp.Sleep(1*time.Second),
	)
}

// WaitForSuccessState waits for the success state to be shown
func (p *LiquidityPoolPage) WaitForSuccessState() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`#success-state`, chromedp.ByID),
	)
}

// GetPairAddress gets the pair address from the success state
func (p *LiquidityPoolPage) GetPairAddress() (string, error) {
	var address string
	err := chromedp.Run(p.ctx,
		chromedp.Text(`#pair-address`, &address, chromedp.ByID),
	)
	return address, err
}

// GetTransactionHash gets the transaction hash from the success state
func (p *LiquidityPoolPage) GetTransactionHash() (string, error) {
	var txHash string
	err := chromedp.Run(p.ctx,
		chromedp.Text(`#transaction-hash`, &txHash, chromedp.ByID),
	)
	return txHash, err
}

// TakeScreenshot takes a screenshot of the current page state
func (p *LiquidityPoolPage) TakeScreenshot(filename string) error {
	var buf []byte
	err := chromedp.Run(p.ctx,
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf, 0644)
}

// LogPageState logs the current page HTML for debugging
func (p *LiquidityPoolPage) LogPageState(prefix string) error {
	var html string
	err := chromedp.Run(p.ctx,
		chromedp.OuterHTML(`html`, &html),
	)
	if err != nil {
		return err
	}

	// Write to debug file
	filename := fmt.Sprintf("debug_%s.html", prefix)
	return ioutil.WriteFile(filename, []byte(html), 0644)
}

// GetTransactionDetails gets the transaction details from the page
func (p *LiquidityPoolPage) GetTransactionDetails() (map[string]string, error) {
	details := make(map[string]string)

	// Try to get various transaction details that might be displayed
	var tokenAddress, tokenAmount, ethAmount string
	chromedp.Run(p.ctx,
		chromedp.Text(`#token-address`, &tokenAddress, chromedp.ByID),
		chromedp.Text(`#token-amount`, &tokenAmount, chromedp.ByID),
		chromedp.Text(`#eth-amount`, &ethAmount, chromedp.ByID),
	)

	if tokenAddress != "" {
		details["token_address"] = tokenAddress
	}
	if tokenAmount != "" {
		details["token_amount"] = tokenAmount
	}
	if ethAmount != "" {
		details["eth_amount"] = ethAmount
	}

	return details, nil
}

// VerifyEmbeddedTransactionData verifies that transaction data is embedded in the page
func (p *LiquidityPoolPage) VerifyEmbeddedTransactionData() error {
	var hasData bool
	err := chromedp.Run(p.ctx,
		chromedp.Evaluate(`
			const sessionData = document.getElementById('session-data');
			sessionData && sessionData.dataset.transactionData ? true : false
		`, &hasData),
	)
	if err != nil {
		return err
	}
	if !hasData {
		return fmt.Errorf("transaction data not embedded in page")
	}
	return nil
}

// WaitForErrorState waits for an error state to be displayed
func (p *LiquidityPoolPage) WaitForErrorState() error {
	return chromedp.Run(p.ctx,
		chromedp.WaitVisible(`.error-message`, chromedp.ByQuery),
	)
}

// GetErrorMessage gets the error message displayed on the page
func (p *LiquidityPoolPage) GetErrorMessage() (string, error) {
	var errorMsg string
	err := chromedp.Run(p.ctx,
		chromedp.Text(`.error-message`, &errorMsg, chromedp.ByQuery),
	)
	return errorMsg, err
}
