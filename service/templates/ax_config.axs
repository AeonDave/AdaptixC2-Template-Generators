/// __NAME_CAP__ service

/// ─── Metadata ──────────────────────────────────────────────────────────────────
let meta = {
    "name":        "__NAME_CAP__",
    "description": "__NAME_CAP__ service plugin",
    "version":     "1.0.0"
};

let serviceName = "__NAME__";

/// ─── Init (called on service load) ─────────────────────────────────────────────

function InitService()
{
    let action = menu.create_action("__NAME_CAP__", function() {
        openServiceDialog();
    });
    menu.add_main_axscript(action);
    /// START CODE HERE — runs when the service is loaded
    /// Example: ax.service_command(serviceName, "load_settings", null);
    /// END CODE HERE
}

/// ─── Service UI ────────────────────────────────────────────────────────────────

function ServiceUI() {}

function openServiceDialog()
{
    let labelInfo = form.create_label("__NAME_CAP__ Service");

    let labelFunction = form.create_label("Function:");
    let comboFunction = form.create_combo();
    comboFunction.addItem("ping");
    /// START CODE HERE — add your functions to the combo
    /// END CODE HERE

    let labelArgs = form.create_label("Arguments:");
    let texteditArgs = form.create_textmulti();
    texteditArgs.setPlaceholder("Optional JSON arguments...");

    let btnRun = form.create_button("Run");

    let layout = form.create_gridlayout();
    layout.addWidget(labelInfo,       0, 0, 1, 4);
    layout.addWidget(labelFunction,   1, 0, 1, 1);
    layout.addWidget(comboFunction,   1, 1, 1, 3);
    layout.addWidget(labelArgs,       2, 0, 1, 1);
    layout.addWidget(texteditArgs,    2, 1, 1, 3);
    layout.addWidget(btnRun,          3, 0, 1, 4);

    form.connect(btnRun, "clicked", function() {
        let fn   = comboFunction.currentText();
        let args = texteditArgs.text();
        ax.service_command(serviceName, fn, args);
    });

    let dialog = form.create_ext_dialog("__NAME_CAP__");
    dialog.setLayout(layout);
    dialog.setSize(480, 320);
    dialog.setButtonsText("Close", "");
    dialog.exec();
}

/// ─── Data handler ──────────────────────────────────────────────────────────────
/// Receives JSON data pushed from the server via TsServiceSendDataClient.

function data_handler(data)
{
    let msg = JSON.parse(data);

    if (!msg.success && msg.error) {
        ax.log("[__NAME_CAP__] Error: " + msg.error);
    } else if (msg.output) {
        ax.log("[__NAME_CAP__] " + msg.output);
    } else {
        ax.log("[__NAME_CAP__] " + JSON.stringify(msg));
    }
}

/// ─── Boot ──────────────────────────────────────────────────────────────────────

ServiceUI();
