/// BeaconAdvanced - ChaCha20-Poly1305 with HMAC-SHA256 Authentication

function ListenerUI(mode_create) {
    let labelHost = form.create_label("Host & port:");
    let comboHost = form.create_combo();
    comboHost.setEnabled(mode_create);
    for (let addr of ax.interfaces()) { comboHost.addItem(addr); }

    let spinPort = form.create_spin();
    spinPort.setRange(1, 65535);
    spinPort.setValue(443);
    spinPort.setEnabled(mode_create);

    let labelCallback = form.create_label("Callback addresses:");
    let textCallback = form.create_list();
    textCallback.setButtonsEnabled(true);
    textCallback.addItem("address:port");

    let labelKey = form.create_label("Encryption key (ChaCha20-256bit):");
    let textKey = form.create_textline(ax.random_string(64, "hex"));
    textKey.setEnabled(mode_create);

    let labelMethod = form.create_label("HTTP Method:");
    let comboMethod = form.create_combo();
    comboMethod.addItems(["POST", "GET"]);

    let labelURI = form.create_label("URIs:");
    let textURI = form.create_list();
    textURI.setButtonsEnabled(true);
    textURI.addItem("/api/v1/data");

    let labelUA = form.create_label("User-Agents:");
    let textUA = form.create_list();
    textUA.setButtonsEnabled(true);
    textUA.addItem("Mozilla/5.0");

    let spinJitter = form.create_spin();
    spinJitter.setRange(0, 10000);
    spinJitter.setValue(0);

    let spinPadMin = form.create_spin();
    spinPadMin.setRange(0, 1000);
    spinPadMin.setValue(0);

    let spinPadMax = form.create_spin();
    spinPadMax.setRange(0, 5000);
    spinPadMax.setValue(0);

    let container = form.create_container();
    container.put("host_bind", comboHost);
    container.put("port_bind", spinPort);
    container.put("callback_addresses", textCallback);
    container.put("encrypt_key", textKey);
    container.put("http_method", comboMethod);
    container.put("uris", textURI);
    container.put("user_agents", textUA);
    container.put("jitter_ms", spinJitter);
    container.put("padding_min", spinPadMin);
    container.put("padding_max", spinPadMax);

    let panel = form.create_panel();
    let layout = form.create_gridlayout();
    layout.addWidget(labelHost, 0, 0);
    layout.addWidget(comboHost, 0, 1);
    layout.addWidget(spinPort, 0, 2);
    layout.addWidget(labelCallback, 1, 0);
    layout.addWidget(textCallback, 1, 1);
    layout.addWidget(labelKey, 2, 0);
    layout.addWidget(textKey, 2, 1);
    layout.addWidget(labelMethod, 3, 0);
    layout.addWidget(comboMethod, 3, 1);
    layout.addWidget(labelURI, 4, 0);
    layout.addWidget(textURI, 4, 1);
    layout.addWidget(labelUA, 5, 0);
    layout.addWidget(textUA, 5, 1);

    panel.setLayout(layout);

    return {
        ui_panel: panel,
        ui_container: container,
        ui_height: 400,
        ui_width: 600
    };
}
