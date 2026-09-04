package web

import "embed"

// Assets embeds the static frontend files
//go:embed index.html style.css app.js icon.png icon.svg favicon.ico
var Assets embed.FS
