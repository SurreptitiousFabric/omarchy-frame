pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Layouts
import qs.Commons
import qs.Ui

Column {
    id: page

    required property var service
    required property color frameForeground
    required property color frameDim
    required property color softFill
    required property string uiFont

    width: parent ? parent.width : 0
    spacing: Style.space(10)
    onVisibleChanged: if (visible && service && !service.galleryLoaded)
        service.loadGallery()

    component SoftButton: Button {
        implicitHeight: 42
        background: page.softFill
        foreground: page.frameForeground
        fontFamily: page.uiFont
        fontSize: Style.font.body
        bordered: false
        focusable: true
    }

    RowLayout {
        width: parent.width
        spacing: Style.space(8)
        SoftButton {
            text: "TV / Art"
            Layout.fillWidth: true
            onClicked: page.service.key("KEY_POWER")
        }
        PanelActionButton {
            iconText: "↻"
            tooltipText: "Refresh artwork"
            fontFamily: page.uiFont
            foreground: page.frameForeground
            size: 42
            focusable: true
            onClicked: page.service.loadGallery()
        }
    }
    Text {
        visible: page.service && !page.service.galleryLoaded
        width: parent.width
        text: "Loading artwork…"
        horizontalAlignment: Text.AlignHCenter
        color: page.frameDim
        font.family: page.uiFont
        font.pixelSize: Style.font.body
    }
    Text {
        visible: page.service && page.service.galleryLoaded && page.service.artGallery.length === 0
        width: parent.width
        text: "No artwork previews available"
        horizontalAlignment: Text.AlignHCenter
        color: page.frameDim
        font.family: page.uiFont
        font.pixelSize: Style.font.body
    }
    GridLayout {
        visible: page.service && page.service.galleryLoaded
        columns: 2
        width: parent.width
        rowSpacing: Style.space(8)
        columnSpacing: Style.space(8)
        Repeater {
            model: page.service ? page.service.artGallery : []
            GalleryCard {
                required property var modelData
                item: modelData
                selectedID: page.service.selectedArtID
                softFill: page.softFill
                uiFont: page.uiFont
                onSelected: id => page.service.selectArt(id)
            }
        }
    }
}
