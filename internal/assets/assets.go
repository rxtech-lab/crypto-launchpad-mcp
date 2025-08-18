package assets

import _ "embed"

//go:embed wallet.js
var WalletJS []byte

//go:embed wallet-connection.js
var WalletConnectionJS []byte

//go:embed deploy-tokens.js
var DeployTokensJS []byte

//go:embed deploy-uniswap.js
var DeployUniswapJS []byte

//go:embed balance-query.js
var BalanceQueryJS []byte

//go:embed deploy.html
var DeployHTML []byte

//go:embed create_pool.html
var CreatePoolHTML []byte

//go:embed generic.html
var GenericHTML []byte

//go:embed deploy_uniswap.html
var DeployUniswapHTML []byte

//go:embed balance.html
var BalanceHTML []byte
