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
    property bool pickerRunning: false
    property bool slideshowOpen: false
    signal uploadRequested
    signal deleteRequested(string id)

    function reset() { slideshowOpen = false }

    width: parent ? parent.width : 0
    spacing: Style.space(10)
    onVisibleChanged: if (visible && service && !service.galleryLoaded)
        service.loadGallery()

    component PageButton: FrameButton {
        foreground: page.frameForeground
        fontFamily: page.uiFont
    }

    RowLayout {
        width: parent.width
        spacing: Style.space(8)
        PageButton {
            text: page.pickerRunning ? "Opening picker…" : "+  Upload photo"
            Layout.fillWidth: true
            enabled: !page.pickerRunning
            onClicked: page.uploadRequested()
        }
        PanelActionButton {
            iconText: "↻"
            tooltipText: "Refresh photos"
            fontFamily: page.uiFont
            foreground: page.frameForeground
            size: Style.space(42)
            focusable: true
            onClicked: page.service.loadGallery()
        }
    }
    PageButton {
        quiet: true
        text: (page.slideshowOpen ? "▾  " : "▸  ") + "Slideshow"
        width: parent.width
        leftAlign: true
        onClicked: page.slideshowOpen = !page.slideshowOpen
    }
    RowLayout {
        visible: page.slideshowOpen
        width: parent.width
        spacing: Style.space(8)
        PageButton {
            text: "Shuffle · 5 min"
            Layout.fillWidth: true
            onClicked: page.service.slideshow(5, true)
        }
        PageButton {
            text: "15 min"
            Layout.fillWidth: true
            onClicked: page.service.slideshow(15, false)
        }
        PageButton {
            quiet: true
            text: "Stop"
            onClicked: page.service.slideshow(0, false)
        }
    }
    Text {
        visible: page.service && !page.service.galleryLoaded
        width: parent.width
        text: "Loading photos…"
        horizontalAlignment: Text.AlignHCenter
        color: page.frameDim
        font.family: page.uiFont
        font.pixelSize: Style.font.body
    }
    Text {
        visible: page.service && page.service.galleryLoaded && page.service.photosGallery.length === 0
        width: parent.width
        text: "No personal photos yet"
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
            model: page.service ? page.service.photosGallery : []
            GalleryCard {
                required property var modelData
                item: modelData
                selectedID: page.service.selectedArtID
                softFill: page.softFill
                uiFont: page.uiFont
                deletable: true
                onSelected: id => page.service.selectArt(id)
                onDeleteRequested: id => page.deleteRequested(id)
            }
        }
    }
}
