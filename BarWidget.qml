pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui

BarWidget {
    id: root

    moduleName: "swa.frame"

    readonly property var service: bar && bar.shell ? bar.shell.serviceFor("swa.frame") : null
    readonly property color panelForeground: bar ? bar.foreground : Color.foreground
    readonly property color panelDim: Qt.darker(panelForeground, 1.45)
    readonly property color softFill: Style.normalFillFor(panelForeground, Color.accent)
    readonly property string uiFont: "sans-serif"
    readonly property string mode: {
        if (!service || !service.snapshot || !service.snapshot.ok)
            return "unknown";
        if (!service.snapshot.online)
            return "off";
        var value = String(service.snapshot.mode || "unknown").toLowerCase();
        return value === "art" || value === "tv" ? value : "unknown";
    }
    readonly property string modeLabel: mode === "art" ? "ART" : mode === "tv" ? "TV" : mode === "off" ? "OFF" : "UNKNOWN"
    readonly property string modeTooltip: mode === "art" ? "Samsung Frame · Art Mode" : mode === "tv" ? "Samsung Frame · watching TV" : mode === "off" ? "Samsung Frame · off" : service && service.snapshot && service.snapshot.ok ? "Samsung Frame · online · mode unavailable" : "Samsung Frame · checking status"

    property bool popupOpen: false
    property string page: "remote"
    property string remotePage: "navigate"
    property bool slideshowOpen: false
    property bool manualSetupOpen: false
    property bool setupCapabilitiesOpen: false
    property string pendingDeleteID: ""

    readonly property bool opened: popupOpen
    readonly property int naturalPanelHeight: {
        if (page === "art" || page === "photos")
            return 620;
        if (page === "setup") {
            var setupHeight = 420;
            if (manualSetupOpen)
                setupHeight += 110;
            if (setupCapabilitiesOpen)
                setupHeight += 170;
            return Math.min(700, setupHeight);
        }
        if (remotePage === "navigate")
            return 520;
        if (remotePage === "media")
            return 450;
        if (remotePage === "tv")
            return 470;
        return 400;
    }

    function close() {
        popupOpen = false;
    }
    function open() {
        popupOpen = true;
    }
    function toggle() {
        popupOpen = !popupOpen;
    }

    component SoftButton: Button {
        implicitHeight: 42
        background: root.softFill
        foreground: root.panelForeground
        fontFamily: root.uiFont
        fontSize: Style.font.body
        bordered: false
        focusable: true
    }

    component QuietButton: Button {
        implicitHeight: 40
        background: "transparent"
        foreground: root.panelForeground
        fontFamily: root.uiFont
        fontSize: Style.font.bodySmall
        bordered: false
        focusable: true
    }

    component RoundButton: PanelActionButton {
        size: 54
        fontFamily: root.uiFont
        fontSize: Style.font.title
        foreground: root.panelForeground
        bordered: false
        focusable: true
    }

    component TvIcon: Item {
        property color stroke: root.panelForeground
        property real iconSize: 24

        implicitWidth: iconSize
        implicitHeight: iconSize

        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height * 0.10
            width: parent.width * 0.82
            height: parent.height * 0.62
            radius: Math.max(1, parent.width * 0.06)
            color: "transparent"
            border.color: parent.stroke
            border.width: Math.max(1, Math.round(parent.width * 0.08))
        }
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height * 0.72
            width: Math.max(1, parent.width * 0.08)
            height: parent.height * 0.12
            color: parent.stroke
        }
        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height * 0.84
            width: parent.width * 0.42
            height: Math.max(1, parent.height * 0.07)
            radius: height / 2
            color: parent.stroke
        }
    }

    component GalleryCard: Rectangle {
        id: card

        required property var item
        property bool deletable: false
        signal selected(string id)
        signal deleteRequested(string id)

        Layout.fillWidth: true
        Layout.preferredHeight: 138
        radius: Style.cornerRadius
        color: root.softFill
        clip: true
        border.width: String(item.id) === root.service.selectedArtID ? 2 : 0
        border.color: Color.accent

        Image {
            anchors.fill: parent
            anchors.margins: card.border.width
            source: "file://" + String(card.item.image)
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
            cache: true
        }

        Rectangle {
            z: 1
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: 46
            gradient: Gradient {
                GradientStop {
                    position: 0
                    color: "transparent"
                }
                GradientStop {
                    position: 1
                    color: Qt.rgba(0, 0, 0, 0.72)
                }
            }
        }

        Rectangle {
            z: 2
            visible: String(card.item.id) === root.service.selectedArtID
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.margins: 9
            width: selectedLabel.implicitWidth + 12
            height: 23
            radius: 12
            color: Color.accent

            Text {
                id: selectedLabel
                anchors.centerIn: parent
                text: "ON TV"
                color: Color.background
                font.family: root.uiFont
                font.pixelSize: Style.font.caption
                font.bold: true
            }
        }

        PanelActionButton {
            z: 3
            visible: card.deletable
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.margins: 7
            size: 30
            iconText: "×"
            tooltipText: "Delete photo"
            fontFamily: root.uiFont
            fontSize: Style.font.title
            foreground: "white"
            hoverColor: Color.urgent
            onClicked: card.deleteRequested(String(card.item.id))
        }

        MouseArea {
            anchors.fill: parent
            z: 0
            cursorShape: Qt.PointingHandCursor
            onClicked: card.selected(String(card.item.id))
        }
    }

    Component {
        id: frameIcon
        TvIcon {
            iconSize: Style.font.display
            stroke: root.service && root.service.snapshot.online ? Color.accent : root.panelDim
        }
    }

    Component {
        id: headerActions
        Row {
            spacing: Style.space(4)
            PanelActionButton {
                iconText: "↻"
                tooltipText: "Refresh"
                fontFamily: root.uiFont
                focusable: true
                foreground: root.panelForeground
                onClicked: root.service && root.service.refresh()
            }
            Button {
                text: root.service && root.service.snapshot.online ? "Off" : "Wake"
                tooltipText: root.service && root.service.snapshot.online ? "Turn TV off" : "Wake TV"
                fontFamily: root.uiFont
                fontSize: Style.font.caption
                focusable: true
                foreground: root.service && root.service.snapshot.online ? root.panelForeground : Color.accent
                background: "transparent"
                bordered: false
                onClicked: root.service && (root.service.snapshot.online ? root.service.key("KEY_POWER") : root.service.wake())
            }
        }
    }

    Process {
        id: photoPicker
        command: ["/usr/bin/zenity", "--file-selection", "--title=Upload a photo to The Frame", "--file-filter=Images | *.jpg *.jpeg *.png"]
        stdout: StdioCollector {
            id: photoPickerOut
            waitForEnd: true
        }
        onExited: function (code) {
            var path = String(photoPickerOut.text || "").trim();
            if (code === 0 && path !== "" && root.service)
                root.service.uploadArt(path);
        }
    }

    onPopupOpenChanged: {
        if (!popupOpen) {
            remotePage = "navigate";
            pendingDeleteID = "";
            slideshowOpen = false;
            manualSetupOpen = false;
            setupCapabilitiesOpen = false;
        }
        if (service) {
            service.pollIntervalMs = Math.max(5, Math.min(120, Number(root.setting("pollSeconds", 15)))) * 1000;
            service.panelOpen = popupOpen;
            if (popupOpen)
                service.refresh();
        }
    }

    implicitWidth: barSize
    implicitHeight: barSize
    opacity: service && service.snapshot.online ? 1 : 0.6

    TvIcon {
        anchors.centerIn: parent
        iconSize: Math.max(14, Style.font.body)
        stroke: root.bar ? root.bar.barForeground : Color.foreground
    }

    WidgetButton {
        id: button
        anchors.fill: parent
        bar: root.bar
        text: " "
        labelVisible: false
        tooltipText: root.modeTooltip
        onPressed: function (mouseButton) {
            if (mouseButton === Qt.MiddleButton && root.service)
                root.service.refresh();
            else
                root.toggle();
        }
    }

    KeyboardPanel {
        id: popup
        anchorItem: button
        bar: root.bar
        owner: root
        open: root.popupOpen
        focusTarget: keyCatcher
        contentWidth: popup.fittedContentWidth(Math.max(410, Math.min(540, Number(root.setting("panelWidth", 440)))))
        contentHeight: popup.cappedContentHeight(Math.min(root.naturalPanelHeight, Math.max(400, Math.min(700, Number(root.setting("panelHeight", 620))))))

        PanelKeyCatcher {
            id: keyCatcher
            anchors.fill: parent
            blocked: deleteConfirm.opened
            onCloseRequested: root.close()
        }

        ColumnLayout {
            id: content
            anchors.fill: parent
            spacing: Style.space(10)

            PanelHero {
                Layout.fillWidth: true
                iconComponent: frameIcon
                title: root.service && root.service.snapshot.device && root.service.snapshot.device.name ? root.service.snapshot.device.name : "Samsung The Frame"
                meta: root.service ? root.service.message : "Connecting"
                detail: root.modeLabel
                foreground: root.panelForeground
                fontFamily: root.uiFont
                trailingControl: headerActions
            }

            Text {
                visible: root.service && root.service.error !== ""
                text: root.service ? root.service.error : ""
                color: Color.urgent
                wrapMode: Text.Wrap
                Layout.fillWidth: true
                font.family: root.uiFont
                font.pixelSize: Style.font.caption
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: Style.space(4)
                Repeater {
                    model: [
                        {
                            label: "Remote",
                            value: "remote"
                        },
                        {
                            label: "Art",
                            value: "art"
                        },
                        {
                            label: "Photos",
                            value: "photos"
                        },
                        {
                            label: "Setup",
                            value: "setup"
                        }
                    ]
                    Button {
                        required property var modelData
                        Layout.fillWidth: true
                        text: modelData.label
                        selected: root.page === modelData.value
                        foreground: root.panelForeground
                        background: "transparent"
                        accent: Color.accent
                        fontFamily: root.uiFont
                        fontSize: Style.font.bodySmall
                        bordered: false
                        focusable: true
                        onClicked: {
                            root.page = modelData.value;
                            if ((root.page === "art" || root.page === "photos") && root.service && !root.service.galleryLoaded)
                                root.service.loadGallery();
                        }
                    }
                }
            }

            PanelSeparator {
                Layout.fillWidth: true
                foreground: root.panelForeground
            }

            ScrollView {
                id: scrollArea
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true
                ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                ScrollBar.vertical.policy: body.implicitHeight > height ? ScrollBar.AsNeeded : ScrollBar.AlwaysOff

                Binding {
                    target: scrollArea.contentItem
                    property: "interactive"
                    value: body.implicitHeight > scrollArea.height
                }

                Column {
                    id: body
                    width: scrollArea.availableWidth
                    spacing: Style.space(12)

                    Column {
                        visible: root.page === "remote"
                        width: parent.width
                        spacing: Style.space(14)

                        RowLayout {
                            width: parent.width
                            spacing: Style.space(4)
                            Repeater {
                                model: [
                                    {
                                        label: "Navigate",
                                        value: "navigate"
                                    },
                                    {
                                        label: "Sound",
                                        value: "sound"
                                    },
                                    {
                                        label: "Media",
                                        value: "media"
                                    },
                                    {
                                        label: "More",
                                        value: "tv"
                                    }
                                ]
                                QuietButton {
                                    required property var modelData
                                    Layout.fillWidth: true
                                    text: modelData.label
                                    selected: root.remotePage === modelData.value
                                    onClicked: root.remotePage = modelData.value
                                }
                            }
                        }

                        Column {
                            visible: root.remotePage === "navigate"
                            width: parent.width
                            spacing: Style.space(12)

                            Rectangle {
                                anchors.horizontalCenter: parent.horizontalCenter
                                width: 222
                                height: 222
                                radius: 111
                                color: root.softFill
                                GridLayout {
                                    anchors.centerIn: parent
                                    columns: 3
                                    rowSpacing: 7
                                    columnSpacing: 7
                                    Item {
                                        Layout.preferredWidth: 54
                                        Layout.preferredHeight: 54
                                    }
                                    RoundButton {
                                        iconText: "▲"
                                        onClicked: root.service.key("KEY_UP")
                                    }
                                    Item {
                                        Layout.preferredWidth: 54
                                        Layout.preferredHeight: 54
                                    }
                                    RoundButton {
                                        iconText: "◀"
                                        onClicked: root.service.key("KEY_LEFT")
                                    }
                                    Button {
                                        Layout.preferredWidth: 58
                                        Layout.preferredHeight: 58
                                        radius: 29
                                        text: "OK"
                                        selected: true
                                        foreground: root.panelForeground
                                        accent: Color.accent
                                        fontFamily: root.uiFont
                                        fontSize: Style.font.body
                                        focusable: true
                                        onClicked: root.service.key("KEY_ENTER")
                                    }
                                    RoundButton {
                                        iconText: "▶"
                                        onClicked: root.service.key("KEY_RIGHT")
                                    }
                                    Item {
                                        Layout.preferredWidth: 54
                                        Layout.preferredHeight: 54
                                    }
                                    RoundButton {
                                        iconText: "▼"
                                        onClicked: root.service.key("KEY_DOWN")
                                    }
                                    Item {
                                        Layout.preferredWidth: 54
                                        Layout.preferredHeight: 54
                                    }
                                }
                            }

                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                SoftButton {
                                    text: "Back"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_RETURN")
                                }
                                SoftButton {
                                    text: "Home"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_HOME")
                                }
                            }
                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                QuietButton {
                                    text: "Source"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_SOURCE")
                                }
                                QuietButton {
                                    text: "Rotate display"
                                    Layout.fillWidth: true
                                    onClicked: root.service.rotate()
                                }
                            }
                        }

                        Column {
                            visible: root.remotePage === "sound"
                            width: parent.width
                            spacing: Style.space(14)
                            PanelSectionHeader {
                                text: "VOLUME"
                                foreground: root.panelForeground
                                fontFamily: root.uiFont
                            }
                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                SoftButton {
                                    text: "−"
                                    Layout.fillWidth: true
                                    fontSize: Style.font.title
                                    onClicked: root.service.key("KEY_VOLDOWN")
                                }
                                SoftButton {
                                    text: "Mute"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_MUTE")
                                }
                                SoftButton {
                                    text: "+"
                                    Layout.fillWidth: true
                                    fontSize: Style.font.title
                                    onClicked: root.service.key("KEY_VOLUP")
                                }
                            }
                            PanelSectionHeader {
                                text: "CHANNEL"
                                foreground: root.panelForeground
                                fontFamily: root.uiFont
                            }
                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                SoftButton {
                                    text: "−"
                                    Layout.fillWidth: true
                                    fontSize: Style.font.title
                                    onClicked: root.service.key("KEY_CHDOWN")
                                }
                                SoftButton {
                                    text: "Guide"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_GUIDE")
                                }
                                SoftButton {
                                    text: "+"
                                    Layout.fillWidth: true
                                    fontSize: Style.font.title
                                    onClicked: root.service.key("KEY_CHUP")
                                }
                            }
                        }

                        Column {
                            visible: root.remotePage === "media"
                            width: parent.width
                            spacing: Style.space(16)
                            Button {
                                anchors.horizontalCenter: parent.horizontalCenter
                                width: 104
                                height: 104
                                radius: 52
                                text: "Play / Pause"
                                selected: true
                                foreground: root.panelForeground
                                accent: Color.accent
                                fontFamily: root.uiFont
                                fontSize: Style.font.bodySmall
                                focusable: true
                                onClicked: root.service.key("KEY_PLAYPAUSE")
                            }
                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                SoftButton {
                                    text: "Previous"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_PREVIOUS")
                                }
                                SoftButton {
                                    text: "Stop"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_STOP")
                                }
                            }
                            RowLayout {
                                width: parent.width
                                spacing: Style.space(8)
                                QuietButton {
                                    text: "Rewind"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_REWIND")
                                }
                                QuietButton {
                                    text: "Fast-forward"
                                    Layout.fillWidth: true
                                    onClicked: root.service.key("KEY_FF")
                                }
                            }
                        }

                        GridLayout {
                            visible: root.remotePage === "tv"
                            width: parent.width
                            columns: 2
                            rowSpacing: Style.space(8)
                            columnSpacing: Style.space(8)
                            Repeater {
                                model: [
                                    {
                                        t: "Channel list",
                                        k: "KEY_CH_LIST"
                                    },
                                    {
                                        t: "Info",
                                        k: "KEY_INFO"
                                    },
                                    {
                                        t: "Menu",
                                        k: "KEY_MENU"
                                    },
                                    {
                                        t: "Tools",
                                        k: "KEY_TOOLS"
                                    },
                                    {
                                        t: "Previous channel",
                                        k: "KEY_PRECH"
                                    },
                                    {
                                        t: "Record",
                                        k: "KEY_REC"
                                    },
                                    {
                                        t: "Captions",
                                        k: "KEY_CAPTION"
                                    },
                                    {
                                        t: "Exit",
                                        k: "KEY_EXIT"
                                    }
                                ]
                                SoftButton {
                                    required property var modelData
                                    Layout.fillWidth: true
                                    text: modelData.t
                                    onClicked: root.service.key(modelData.k)
                                }
                            }
                        }
                    }

                    Column {
                        visible: root.page === "art"
                        width: parent.width
                        spacing: Style.space(10)
                        onVisibleChanged: if (visible && root.service && !root.service.galleryLoaded)
                            root.service.loadGallery()
                        RowLayout {
                            width: parent.width
                            spacing: Style.space(8)
                            SoftButton {
                                text: "Enter Art Mode"
                                Layout.fillWidth: true
                                onClicked: root.service.key("KEY_AMBIENT")
                            }
                            PanelActionButton {
                                iconText: "↻"
                                tooltipText: "Refresh artwork"
                                fontFamily: root.uiFont
                                foreground: root.panelForeground
                                size: 42
                                focusable: true
                                onClicked: root.service.loadGallery()
                            }
                        }
                        Text {
                            visible: root.service && !root.service.galleryLoaded
                            width: parent.width
                            text: "Loading artwork…"
                            horizontalAlignment: Text.AlignHCenter
                            color: root.panelDim
                            font.family: root.uiFont
                            font.pixelSize: Style.font.body
                        }
                        Text {
                            visible: root.service && root.service.galleryLoaded && root.service.artGallery.length === 0
                            width: parent.width
                            text: "No artwork previews available"
                            horizontalAlignment: Text.AlignHCenter
                            color: root.panelDim
                            font.family: root.uiFont
                            font.pixelSize: Style.font.body
                        }
                        GridLayout {
                            visible: root.service && root.service.galleryLoaded
                            columns: 2
                            width: parent.width
                            rowSpacing: Style.space(8)
                            columnSpacing: Style.space(8)
                            Repeater {
                                model: root.service ? root.service.artGallery : []
                                GalleryCard {
                                    required property var modelData
                                    item: modelData
                                    onSelected: id => root.service.selectArt(id)
                                }
                            }
                        }
                    }

                    Column {
                        visible: root.page === "photos"
                        width: parent.width
                        spacing: Style.space(10)
                        onVisibleChanged: if (visible && root.service && !root.service.galleryLoaded)
                            root.service.loadGallery()
                        RowLayout {
                            width: parent.width
                            spacing: Style.space(8)
                            SoftButton {
                                text: photoPicker.running ? "Opening picker…" : "+  Upload photo"
                                Layout.fillWidth: true
                                enabled: !photoPicker.running
                                onClicked: photoPicker.running = true
                            }
                            PanelActionButton {
                                iconText: "↻"
                                tooltipText: "Refresh photos"
                                fontFamily: root.uiFont
                                foreground: root.panelForeground
                                size: 42
                                focusable: true
                                onClicked: root.service.loadGallery()
                            }
                        }
                        QuietButton {
                            text: (root.slideshowOpen ? "▾  " : "▸  ") + "Slideshow"
                            width: parent.width
                            leftAlign: true
                            onClicked: root.slideshowOpen = !root.slideshowOpen
                        }
                        RowLayout {
                            visible: root.slideshowOpen
                            width: parent.width
                            spacing: Style.space(8)
                            SoftButton {
                                text: "Shuffle · 5 min"
                                Layout.fillWidth: true
                                onClicked: root.service.slideshow(5, true)
                            }
                            SoftButton {
                                text: "15 min"
                                Layout.fillWidth: true
                                onClicked: root.service.slideshow(15, false)
                            }
                            QuietButton {
                                text: "Stop"
                                onClicked: root.service.slideshow(0, false)
                            }
                        }
                        Text {
                            visible: root.service && !root.service.galleryLoaded
                            width: parent.width
                            text: "Loading photos…"
                            horizontalAlignment: Text.AlignHCenter
                            color: root.panelDim
                            font.family: root.uiFont
                            font.pixelSize: Style.font.body
                        }
                        Text {
                            visible: root.service && root.service.galleryLoaded && root.service.photosGallery.length === 0
                            width: parent.width
                            text: "No personal photos yet"
                            horizontalAlignment: Text.AlignHCenter
                            color: root.panelDim
                            font.family: root.uiFont
                            font.pixelSize: Style.font.body
                        }
                        GridLayout {
                            visible: root.service && root.service.galleryLoaded
                            columns: 2
                            width: parent.width
                            rowSpacing: Style.space(8)
                            columnSpacing: Style.space(8)
                            Repeater {
                                model: root.service ? root.service.photosGallery : []
                                GalleryCard {
                                    required property var modelData
                                    item: modelData
                                    deletable: true
                                    onSelected: id => root.service.selectArt(id)
                                    onDeleteRequested: id => root.pendingDeleteID = id
                                }
                            }
                        }
                    }

                    Column {
                        visible: root.page === "setup"
                        width: parent.width
                        spacing: Style.space(10)
                        PanelSectionHeader {
                            text: "TV CONNECTION"
                            foreground: root.panelForeground
                            fontFamily: root.uiFont
                        }
                        SoftButton {
                            text: "Find nearby TVs"
                            width: parent.width
                            onClicked: root.service.discover()
                        }
                        Repeater {
                            model: root.service ? root.service.devices : []
                            SoftButton {
                                required property var modelData
                                width: parent.width
                                text: String(modelData.name || modelData.model || modelData.ip)
                                onClicked: root.service.configure(String(modelData.ip))
                            }
                        }
                        QuietButton {
                            text: (root.manualSetupOpen ? "▾  " : "▸  ") + "Enter address manually"
                            width: parent.width
                            leftAlign: true
                            onClicked: root.manualSetupOpen = !root.manualSetupOpen
                        }
                        Column {
                            visible: root.manualSetupOpen
                            width: parent.width
                            spacing: Style.space(8)
                            TextField {
                                id: ipField
                                width: parent.width
                                placeholderText: "TV IP address"
                                inputMethodHints: Qt.ImhFormattedNumbersOnly
                                font.family: root.uiFont
                            }
                            SoftButton {
                                text: "Save address"
                                width: parent.width
                                enabled: ipField.text.trim() !== ""
                                onClicked: root.service.configure(ipField.text)
                            }
                        }
                        PanelSeparator {
                            width: parent.width
                            foreground: root.panelForeground
                        }
                        PanelSectionHeader {
                            text: "ROTATING STAND"
                            foreground: root.panelForeground
                            fontFamily: root.uiFont
                        }
                        Text {
                            width: parent.width
                            text: "Pair the stand once with a compatible Samsung Smart Remote. Rotation then works from Navigate."
                            wrapMode: Text.Wrap
                            color: root.panelDim
                            font.family: root.uiFont
                            font.pixelSize: Style.font.bodySmall
                        }
                        QuietButton {
                            text: (root.setupCapabilitiesOpen ? "▾  " : "▸  ") + "Technical capabilities"
                            width: parent.width
                            leftAlign: true
                            onClicked: root.setupCapabilitiesOpen = !root.setupCapabilitiesOpen
                        }
                        Column {
                            visible: root.setupCapabilitiesOpen
                            width: parent.width
                            spacing: Style.space(9)
                            Repeater {
                                model: root.service ? root.service.capabilities : []
                                Column {
                                    required property var modelData
                                    width: parent.width
                                    spacing: 2
                                    Text {
                                        text: String(parent.modelData.group || "")
                                        font.bold: true
                                        font.family: root.uiFont
                                        font.pixelSize: Style.font.bodySmall
                                        color: root.panelForeground
                                    }
                                    Text {
                                        text: String(parent.modelData.items || "")
                                        width: parent.width
                                        wrapMode: Text.Wrap
                                        font.family: root.uiFont
                                        font.pixelSize: Style.font.caption
                                        color: root.panelDim
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        ConfirmDialog {
            id: deleteConfirm
            z: 100
            anchors.fill: parent
            opened: root.pendingDeleteID !== ""
            message: "Delete this photo from The Frame?"
            cancelText: "Keep"
            confirmText: "Delete"
            foreground: root.panelForeground
            fontFamily: root.uiFont
            onCanceled: root.pendingDeleteID = ""
            onConfirmed: {
                var id = root.pendingDeleteID;
                root.pendingDeleteID = "";
                if (root.service)
                    root.service.deleteArt(id);
            }
        }
    }
}
