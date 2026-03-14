/// __NAME_CAP__ service

function ServiceUI()
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

    let layout = form.create_gridlayout();
    layout.addWidget(labelInfo,       0, 0, 1, 4);
    layout.addWidget(labelFunction,   1, 0, 1, 1);
    layout.addWidget(comboFunction,   1, 1, 1, 3);
    layout.addWidget(labelArgs,       2, 0, 1, 1);
    layout.addWidget(texteditArgs,    2, 1, 1, 3);

    form.set_layout(layout);

    form.on_save(function() {
        return JSON.stringify({
            "function": comboFunction.currentText(),
            "args":     texteditArgs.toPlainText()
        });
    });
}

ServiceUI();
