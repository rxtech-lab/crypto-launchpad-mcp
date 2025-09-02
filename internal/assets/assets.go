package assets

import _ "embed"

//go:embed signing.html
var SigningHTML []byte

//go:embed signing_app.js
var SigningAppJS []byte

//go:embed signing_app.css
var SigningAppCSS []byte

//go:embed error.html
var ErrorHTML []byte
