# CARVION Studio Desktop Host

This directory contains the Wails v3 application module and its React frontend.
Run development and verification commands from the repository root; setup details
are documented in [`../docs/DESKTOP.md`](../docs/DESKTOP.md).

The Go host calls the root CARVION Core in process through generated Wails
bindings. It does not use the HTTP API as a bridge.
