# Service Template Generator

Generates a ready-to-build AdaptixC2 **service** extender plugin — optionally
including a **post-build wrapper pipeline** for payload transformation.

A service plugin runs server-side and exposes callable functions to operators.
Unlike agents (implant + plugin) or listeners (transport + plugin), services
are pure server-side logic — think notifications, integrations, or utilities.

A **wrapper** is a specialized service that hooks into `agent.generate` (post-phase)
and applies configurable pipeline stages (encrypt, pack, obfuscate, …) to the
generated payload before it reaches the operator.

## Quick start

```powershell
# Plain service
.\generator.ps1 -Name telegram

# Service with wrapper pipeline
.\generator.ps1 -Name crystalpalace -Wrapper

# Bash — plain service
NAME=telegram ./generator.sh

# Bash — with wrapper pipeline
NAME=crystalpalace WRAPPER=1 ./generator.sh
```

When neither `-Wrapper` nor `WRAPPER=1` is set, the generator asks interactively:

```
Include post-build wrapper pipeline? [y/N]:
```

## What gets generated

**Plain service:**

```
<name>_service/
├── config.yaml          # Service manifest (extender_type: "service")
├── go.mod               # Go module
├── Makefile             # Build targets
├── pl_main.go           # Plugin entry + Call handler
└── ax_config.axs        # Service UI form
```

**With wrapper pipeline (`-Wrapper`):**

```
<name>_wrapper/
├── config.yaml          # Service manifest (extender_type: "service")
├── go.mod               # Go module
├── Makefile             # Build targets
├── pl_main.go           # Plugin entry + event hook + Call handler
├── pl_wrapper.go        # Pipeline engine (Stage, RunPipeline)
└── ax_config.axs        # Wrapper UI (status, config save/load)
```

## Framework interface

Both variants implement `adaptix.PluginService`:

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

## Implementing wrapper stages

When the wrapper pipeline is included, `pl_main.go` contains an `initStages()`
function where you register transformation stages:

```go
func initStages() {
    RegisterStage(Stage{
        Name:    "rdll_loader",
        Enabled: true,
        Run:     stageRdllLoader,
    })
}
```

Each stage function has the signature:

```go
func(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error)
```

Stages run in registration order. If any stage returns an error, the pipeline
stops and the original payload is returned. See the root README for detailed
pipeline documentation and examples.

## Modular addons

The wrapper is the first **addon**. The generator checks the addon subdirectory
first for each template file -- if the addon provides an override, it is used;
otherwise the base template is used. Future addons (alerting, C2 bridges, etc.)
follow the same pattern under `service/templates/<addon>/`.

## Placeholders

| Placeholder     | Replaced with                 |
|-----------------|-------------------------------|
| `__NAME__`      | Lowercase service name        |
| `__NAME_CAP__`  | Capitalized service name      |
