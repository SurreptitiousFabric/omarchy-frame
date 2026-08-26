pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import qs.Commons
import qs.Ui

Column {
    id: page

    required property var service
    required property color frameForeground
    required property color frameDim
    required property string uiFont
    property bool manualSetupOpen: false
    property bool capabilitiesOpen: false
    readonly property bool editorActive: ipField.activeFocus

    function reset() {
        manualSetupOpen = false
        capabilitiesOpen = false
    }

    width: parent ? parent.width : 0
    spacing: Style.space(10)

    component PageButton: FrameButton {
        foreground: page.frameForeground
        fontFamily: page.uiFont
    }

    PanelSectionHeader { text: "TV CONNECTION"; foreground: page.frameForeground; fontFamily: page.uiFont }
    PageButton {
        text: "Find nearby TVs"
        width: parent.width
        onClicked: page.service.discover()
    }
    Repeater {
        model: page.service ? page.service.devices : []
        PageButton {
            required property var modelData
            width: parent.width
            text: String(modelData.name || modelData.model || modelData.ip)
            onClicked: page.service.configure(String(modelData.ip))
        }
    }
    PageButton {
        quiet: true
        text: (page.manualSetupOpen ? "▾  " : "▸  ") + "Enter address manually"
        width: parent.width
        leftAlign: true
        onClicked: page.manualSetupOpen = !page.manualSetupOpen
    }
    Column {
        visible: page.manualSetupOpen
        width: parent.width
        spacing: Style.space(8)
        TextField {
            id: ipField
            width: parent.width
            placeholderText: "TV IP address"
            inputMethodHints: Qt.ImhFormattedNumbersOnly
            font.family: page.uiFont
        }
        PageButton {
            text: "Save address"
            width: parent.width
            enabled: ipField.text.trim() !== ""
            onClicked: page.service.configure(ipField.text)
        }
    }
    PanelSeparator { width: parent.width; foreground: page.frameForeground }
    PanelSectionHeader { text: "ROTATING STAND"; foreground: page.frameForeground; fontFamily: page.uiFont }
    Text {
        width: parent.width
        text: "Pair the stand once with a compatible Samsung Smart Remote. Rotation then works from Navigate."
        wrapMode: Text.Wrap
        color: page.frameDim
        font.family: page.uiFont
        font.pixelSize: Style.font.bodySmall
    }
    PageButton {
        quiet: true
        text: (page.capabilitiesOpen ? "▾  " : "▸  ") + "Technical capabilities"
        width: parent.width
        leftAlign: true
        onClicked: page.capabilitiesOpen = !page.capabilitiesOpen
    }
    Column {
        visible: page.capabilitiesOpen
        width: parent.width
        spacing: Style.space(9)
        Repeater {
            model: page.service ? page.service.capabilities : []
            Column {
                required property var modelData
                width: parent.width
                spacing: Style.space(2)
                Text {
                    text: String(parent.modelData.group || "")
                    font.bold: true
                    font.family: page.uiFont
                    font.pixelSize: Style.font.bodySmall
                    color: page.frameForeground
                }
                Text {
                    text: String(parent.modelData.items || "")
                    width: parent.width
                    wrapMode: Text.Wrap
                    font.family: page.uiFont
                    font.pixelSize: Style.font.caption
                    color: page.frameDim
                }
            }
        }
    }
}
