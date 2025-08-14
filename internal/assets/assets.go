package assets

import _ "embed"

//go:embed wallet.js
var WalletJS []byte

//go:embed deploy.html
var DeployHTML []byte

//go:embed create_pool.html
var CreatePoolHTML []byte

//go:embed generic.html
var GenericHTML []byte
