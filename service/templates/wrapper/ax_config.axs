/// __NAME_CAP__ wrapper service

/// ─── Metadata ──────────────────────────────────────────────────────────────────
let meta = {
    "name":        "__NAME_CAP__",
    "description": "__NAME_CAP__ post-build wrapper pipeline",
    "version":     "1.0.0"
};

let serviceName = "__NAME__";

/// ─── Init (called on service load) ─────────────────────────────────────────────

function InitService()
{
    ax.service_command(serviceName, "load_config", null);
}

/// ─── Service UI ────────────────────────────────────────────────────────────────

function ServiceUI()
{
    let labelInfo = form.create_label("__NAME_CAP__ Wrapper");

    let labelFunction = form.create_label("Action:");
    let comboFunction = form.create_combo();
    comboFunction.addItem("status");
    comboFunction.addItem("save_config");
    comboFunction.addItem("load_config");
    /// START CODE HERE — add your functions to the combo
    /// END CODE HERE

    let labelArgs = form.create_label("Arguments:");
    let texteditArgs = form.create_textmulti();
    texteditArgs.setPlaceholder("JSON config or arguments...");

    let layout = form.create_gridlayout();
    layout.addWidget(labelInfo,       0, 0, 1, 4);
    layout.addWidget(labelFunction,   1, 0, 1, 1);
    layout.addWidget(comboFunction,   1, 1, 1, 3);
    layout.addWidget(labelArgs,       2, 0, 1, 1);
    layout.addWidget(texteditArgs,    2, 1, 1, 3);

    form.set_layout(layout);

    form.on_save(function() {
        let fn   = comboFunction.currentText();
        let args = texteditArgs.toPlainText();
        ax.service_command(serviceName, fn, args);
    });
}

/// ─── Data handler ──────────────────────────────────────────────────────────────

function data_handler(data)
{
    let msg = JSON.parse(data);

    if (msg.action === "error" || (!msg.success && msg.error)) {
        ax.log("[__NAME_CAP__] Error: " + (msg.error || JSON.stringify(msg)));
    } else if (msg.action === "status") {
        ax.log("[__NAME_CAP__] " + msg.output);
    } else if (msg.action === "save_config") {
        ax.log("[__NAME_CAP__] " + msg.output);
    } else if (msg.action === "load_config") {
        ax.log("[__NAME_CAP__] Config loaded: " + msg.output);
    } else {
        ax.log("[__NAME_CAP__] " + JSON.stringify(msg));
    }
}

/// ─── Init ──────────────────────────────────────────────────────────────────────

ServiceUI();
InitService();
