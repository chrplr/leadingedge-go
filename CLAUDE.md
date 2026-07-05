# Leading Edge (Go port) — notes for Claude

A Go re-implementation of the *Code the Classics* game **Leading Edge**, built on the
[pgzgo](https://github.com/chrplr/pgzgo) harness over
[go-sdl3](https://github.com/Zyko0/go-sdl3). Single module, sources at the repo root.

## Build & run

- Native: `go run .` — go-sdl3 bundles the SDL3 libraries (purego, no CGo, no system
  SDL). Assets are embedded, so the binary is self-contained.
- Headless logic check: `go run . -selftest`.
- Keep `go.mod` on the **published** `go-sdl3` and a pinned `pgzgo` version so
  `go build` / `go get` work for everyone. The browser toolchain is applied only in CI
  (below).

## Browser build (WebAssembly)

Deployed to GitHub Pages on every push to `main` by
`.github/workflows/pages.yml`. The browser build needs SDL3-on-Emscripten plus js
bindings not yet upstream, so the workflow:

1. checks out the [chrplr/go-sdl3-wasm](https://github.com/chrplr/go-sdl3-wasm) fork
   (branch `wasm-render-fixes`) as a sibling,
2. runs `go mod edit -replace github.com/Zyko0/go-sdl3=../go-sdl3-wasm` **in CI only**,
3. builds + bundles with the fork's `wasmsdl` tool, using `web/index.html` as the shell.

**Never commit that `replace`**, and don't commit `*.wasm` or `dist/`. To build locally,
clone the fork as a sibling and apply the same replace, then
`(cd ../go-sdl3-wasm && go run ./cmd/wasmsdl serve -html "$OLDPWD/web/index.html" "$OLDPWD")`.

## Audio

Audio is **bespoke** (`audio.go`, its own SDL3_mixer wrapper) because the game needs behaviour pgzgo doesn't offer; `Config.Audio` is deliberately `nil`. Do not re-add a `runtime.GOOS=="js"` guard — the mixer works in the browser now. Sound works in the browser as of `pgzgo v0.3.0` + the fork's SDL3_mixer
bindings; the `AudioContext` unlocks on the first keypress (title music is silent
until the game is started).
