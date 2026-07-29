package webui

import "embed"

// Assets contains the Vite production build. The tracked placeholder keeps
// backend-only tests buildable before the frontend is compiled.
//
//go:embed all:dist
var Assets embed.FS
