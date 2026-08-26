pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui

Column {
    id: page

    required property var service
    required property color frameForeground
    required property color softFill
    required property string uiFont
    property string section: "navigate"

    function reset() { section = "navigate" }

    width: parent ? parent.width : 0
    spacing: Style.space(14)

    component PageButton: FrameButton {
        foreground: page.frameForeground
        fontFamily: page.uiFont
    }
    component RoundButton: PanelActionButton {
        size: Style.space(54)
        fontFamily: page.uiFont
        fontSize: Style.font.title
        foreground: page.frameForeground
        bordered: false
        focusable: true
    }

    FrameTabGroup {
        width: parent.width
        options: [
            { label: "Navigate", value: "navigate" },
            { label: "Sound", value: "sound" },
            { label: "Media", value: "media" },
            { label: "More", value: "tv" }
        ]
        value: page.section
        foreground: page.frameForeground
        background: "transparent"
        fontFamily: page.uiFont
        fontSize: Style.font.bodySmall
        onChanged: value => page.section = value
    }

    Column {
        visible: page.section === "navigate"
        width: parent.width
        spacing: Style.space(12)

        Rectangle {
            anchors.horizontalCenter: parent.horizontalCenter
            width: Style.space(222)
            height: width
            radius: width / 2
            color: page.softFill
            GridLayout {
                anchors.centerIn: parent
                columns: 3
                rowSpacing: Style.space(7)
                columnSpacing: Style.space(7)
                Item { Layout.preferredWidth: Style.space(54); Layout.preferredHeight: Style.space(54) }
                RoundButton { iconText: "▲"; onClicked: page.service.key("KEY_UP") }
                Item { Layout.preferredWidth: Style.space(54); Layout.preferredHeight: Style.space(54) }
                RoundButton { iconText: "◀"; onClicked: page.service.key("KEY_LEFT") }
                Button {
                    Layout.preferredWidth: Style.space(58)
                    Layout.preferredHeight: Style.space(58)
                    radius: width / 2
                    text: "OK"
                    selected: true
                    foreground: page.frameForeground
                    accent: Color.accent
                    fontFamily: page.uiFont
                    fontSize: Style.font.body
                    focusable: true
                    onClicked: page.service.key("KEY_ENTER")
                }
                RoundButton { iconText: "▶"; onClicked: page.service.key("KEY_RIGHT") }
                Item { Layout.preferredWidth: Style.space(54); Layout.preferredHeight: Style.space(54) }
                RoundButton { iconText: "▼"; onClicked: page.service.key("KEY_DOWN") }
                Item { Layout.preferredWidth: Style.space(54); Layout.preferredHeight: Style.space(54) }
            }
        }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            PageButton { text: "Back"; Layout.fillWidth: true; onClicked: page.service.key("KEY_RETURN") }
            PageButton { text: "Home"; Layout.fillWidth: true; onClicked: page.service.key("KEY_HOME") }
        }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            PageButton { quiet: true; text: "Source"; Layout.fillWidth: true; onClicked: page.service.key("KEY_SOURCE") }
            PageButton { quiet: true; text: "Rotate display"; Layout.fillWidth: true; onClicked: page.service.rotate() }
        }
    }

    Column {
        visible: page.section === "sound"
        width: parent.width
        spacing: Style.space(14)
        PanelSectionHeader { text: "VOLUME"; foreground: page.frameForeground; fontFamily: page.uiFont }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            Repeater {
                model: [
                    { label: "−", key: "KEY_VOLDOWN" },
                    { label: "Mute", key: "KEY_MUTE" },
                    { label: "+", key: "KEY_VOLUP" }
                ]
                PageButton {
                    required property var modelData
                    Layout.fillWidth: true
                    text: modelData.label
                    fontSize: modelData.label === "Mute" ? Style.font.body : Style.font.title
                    onClicked: page.service.key(modelData.key)
                }
            }
        }
        PanelSectionHeader { text: "CHANNEL"; foreground: page.frameForeground; fontFamily: page.uiFont }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            Repeater {
                model: [
                    { label: "−", key: "KEY_CHDOWN" },
                    { label: "Guide", key: "KEY_GUIDE" },
                    { label: "+", key: "KEY_CHUP" }
                ]
                PageButton {
                    required property var modelData
                    Layout.fillWidth: true
                    text: modelData.label
                    fontSize: modelData.label === "Guide" ? Style.font.body : Style.font.title
                    onClicked: page.service.key(modelData.key)
                }
            }
        }
    }

    Column {
        visible: page.section === "media"
        width: parent.width
        spacing: Style.space(16)
        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            width: Style.space(104)
            height: width
            radius: width / 2
            text: "Play / Pause"
            selected: true
            foreground: page.frameForeground
            accent: Color.accent
            fontFamily: page.uiFont
            fontSize: Style.font.bodySmall
            focusable: true
            onClicked: page.service.key("KEY_PLAYPAUSE")
        }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            PageButton { text: "Previous"; Layout.fillWidth: true; onClicked: page.service.key("KEY_PREVIOUS") }
            PageButton { text: "Stop"; Layout.fillWidth: true; onClicked: page.service.key("KEY_STOP") }
        }
        RowLayout {
            width: parent.width
            spacing: Style.space(8)
            PageButton { quiet: true; text: "Rewind"; Layout.fillWidth: true; onClicked: page.service.key("KEY_REWIND") }
            PageButton { quiet: true; text: "Fast-forward"; Layout.fillWidth: true; onClicked: page.service.key("KEY_FF") }
        }
    }

    GridLayout {
        visible: page.section === "tv"
        width: parent.width
        columns: 2
        rowSpacing: Style.space(8)
        columnSpacing: Style.space(8)
        Repeater {
            model: [
                { label: "Channel list", key: "KEY_CH_LIST" },
                { label: "Info", key: "KEY_INFO" },
                { label: "Menu", key: "KEY_MENU" },
                { label: "Tools", key: "KEY_TOOLS" },
                { label: "Previous channel", key: "KEY_PRECH" },
                { label: "Record", key: "KEY_REC" },
                { label: "Captions", key: "KEY_CAPTION" },
                { label: "Exit", key: "KEY_EXIT" }
            ]
            PageButton {
                required property var modelData
                Layout.fillWidth: true
                text: modelData.label
                onClicked: page.service.key(modelData.key)
            }
        }
    }
}
