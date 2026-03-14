# Service Template Generator

Generates a ready-to-build AdaptixC2 **service** extender plugin.

A service plugin runs server-side and exposes callable functions to operators.
Unlike agents (implant + plugin) or listeners (transport + plugin), services
are pure server-side logic — think notifications, integrations, or utilities.

## Quick start

```powershell
# PowerShell
.\generator.ps1 -Name telegram

# Bash
NAME=telegram ./generator.sh
```

## What gets generated

```
<name>_service/
├── config.yaml          # Service manifest (extender_type: "service")
├── go.mod               # Go module
├── Makefile             # Build targets
├── pl_main.go           # Plugin entry + Call handler
└── ax_config.axs        # Service UI form
```

## Framework interface

The service plugin implements `adaptix.PluginService`:

```go
type PluginService interface {
    Call(operator string, function string, args string)
}
```

The `Call` method is invoked when an operator triggers a service function from
the Adaptix UI.  `function` identifies which action to run, and `args` carries
the JSON-encoded arguments from the form defined in `ax_config.axs`.

## Implementing your service

1. **Add functions** — extend the `switch` in `pl_main.go → Call()`.
2. **Add UI controls** — edit `ax_config.axs` to add your function names to the
   combo box and any extra form fields.
3. **Add Teamserver methods** — expand the `Teamserver` interface with any methods
   your service needs (see the agent/listener templates for the full set).
4. **Build** — `go mod tidy && make plugin`.
5. **Deploy** — copy `dist/` contents to `<adaptix-server>/extenders/`.

## Placeholders

| Placeholder     | Replaced with                 |
|-----------------|-------------------------------|
| `__NAME__`      | Lowercase service name        |
| `__NAME_CAP__`  | Capitalized service name      |
