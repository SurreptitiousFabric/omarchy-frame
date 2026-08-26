pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui
import "components"

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
            return "offline";
        var value = String(service.snapshot.mode || "unknown").toLowerCase();
        return value === "art" || value === "tv" ? value : "unknown";
    }
    readonly property string modeLabel: mode === "art" ? "ART" : mode === "tv" ? "TV" : mode === "offline" ? "OFFLINE" : "ART / TV?"
    readonly property string modeTooltip: mode === "art" ? "Samsung Frame · Art Mode" : mode === "tv" ? "Samsung Frame · watching TV" : mode === "offline" ? "Samsung Frame · unreachable" : service && service.snapshot && service.snapshot.ok ? "Samsung Frame · online · Samsung did not report a reliable mode" : "Samsung Frame · checking status"

    property bool popupOpen: false
    property string page: "remote"
    property string pendingDeleteID: ""

    readonly property bool opened: popupOpen
    readonly property int naturalPanelHeight: {
        if (page === "art" || page === "photos")
            return 620;
        if (page === "setup") {
            var setupHeight = 420;
            if (setupPage.manualSetupOpen)
                setupHeight += 110;
            if (setupPage.capabilitiesOpen)
                setupHeight += 170;
            return Math.min(700, setupHeight);
        }
        if (remotePageView.section === "navigate")
            return 520;
        if (remotePageView.section === "media")
            return 450;
        if (remotePageView.section === "tv")
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

    Component {
        id: frameIcon
        FrameTvIcon {
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
                onClicked: root.service && (root.service.snapshot.online ? root.service.powerOff() : root.service.wake())
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
            remotePageView.reset();
            pendingDeleteID = "";
            photosPage.reset();
            setupPage.reset();
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

    FrameTvIcon {
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

                    RemotePage {
                        id: remotePageView
                        visible: root.page === "remote"
                        service: root.service
                        frameForeground: root.panelForeground
                        softFill: root.softFill
                        uiFont: root.uiFont

                    }

                    ArtPage {
                        visible: root.page === "art"
                        service: root.service
                        frameForeground: root.panelForeground
                        frameDim: root.panelDim
                        softFill: root.softFill
                        uiFont: root.uiFont
                    }

                    PhotosPage {
                        id: photosPage
                        visible: root.page === "photos"
                        service: root.service
                        frameForeground: root.panelForeground
                        frameDim: root.panelDim
                        softFill: root.softFill
                        uiFont: root.uiFont
                        pickerRunning: photoPicker.running
                        onUploadRequested: photoPicker.running = true
                        onDeleteRequested: id => root.pendingDeleteID = id
                    }

                    SetupPage {
                        id: setupPage
                        visible: root.page === "setup"
                        service: root.service
                        frameForeground: root.panelForeground
                        frameDim: root.panelDim
                        softFill: root.softFill
                        uiFont: root.uiFont
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
