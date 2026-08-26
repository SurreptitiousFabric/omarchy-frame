import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui

Rectangle {
    id: card

    required property var item
    required property string selectedID
    required property color softFill
    required property string uiFont
    property bool deletable: false
    signal selected(string id)
    signal deleteRequested(string id)

    function activate() { selected(String(item.id)) }

    Layout.fillWidth: true
    Layout.preferredHeight: Style.space(138)
    activeFocusOnTab: true
    radius: Style.cornerRadius
    color: softFill
    clip: true
    border.width: activeFocus || String(item.id) === selectedID ? Style.space(2) : 0
    border.color: Color.accent
    Accessible.role: Accessible.Button
    Accessible.name: deletable ? "Select photo" : "Select artwork"
    Accessible.description: String(item.id) === selectedID ? "Currently displayed on the television" : "Display this item on the television"

    Keys.onReturnPressed: event => {
        card.activate();
        event.accepted = true;
    }
    Keys.onEnterPressed: event => {
        card.activate();
        event.accepted = true;
    }
    Keys.onSpacePressed: event => {
        card.activate();
        event.accepted = true;
    }

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
        height: Style.space(46)
        gradient: Gradient {
            GradientStop { position: 0; color: "transparent" }
            GradientStop { position: 1; color: Qt.rgba(0, 0, 0, 0.72) }
        }
    }

    Rectangle {
        z: 2
        visible: String(card.item.id) === card.selectedID
        anchors.left: parent.left
        anchors.bottom: parent.bottom
        anchors.margins: Style.space(9)
        width: selectedLabel.implicitWidth + Style.space(12)
        height: Style.space(23)
        radius: height / 2
        color: Color.accent

        Text {
            id: selectedLabel
            anchors.centerIn: parent
            text: "ON TV"
            color: Color.background
            font.family: card.uiFont
            font.pixelSize: Style.font.caption
            font.bold: true
        }
    }

    PanelActionButton {
        z: 3
        visible: card.deletable
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: Style.space(7)
        size: Style.space(30)
        iconText: "×"
        tooltipText: "Delete photo"
        fontFamily: card.uiFont
        fontSize: Style.font.title
        foreground: "white"
        hoverColor: Color.urgent
        focusable: true
        onClicked: card.deleteRequested(String(card.item.id))
    }

    MouseArea {
        anchors.fill: parent
        z: 0
        cursorShape: Qt.PointingHandCursor
        onClicked: {
            card.forceActiveFocus();
            card.selected(String(card.item.id));
        }
    }
}
