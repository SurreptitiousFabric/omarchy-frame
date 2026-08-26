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
  readonly property color panelDim: Qt.darker(panelForeground, 1.5)
  readonly property string uiFont: "sans-serif"
  property bool popupOpen: false
  property string page: "remote"
  property string remotePage: "navigate"
  property bool setupCapabilitiesOpen: false
  property string pendingDeleteID: ""
  function close() { popupOpen = false }
  function open() { popupOpen = true }
  function toggle() { popupOpen = !popupOpen }
  readonly property bool opened: popupOpen

  component ActionButton: Button {
    implicitHeight: 42; fontFamily: root.uiFont; fontSize: Style.font.body
    foreground: root.panelForeground; background: "transparent"
    bordered: true; focusable: true
  }
  component NavButton: PanelActionButton {
    size: 52; fontFamily: root.uiFont; fontSize: Style.font.subtitle
    foreground: root.panelForeground; bordered: true; focusable: true
  }
  component GalleryCard: Rectangle {
    required property var item
    property bool deletable: false
    signal selected(string id)
    signal deleteRequested(string id)
    Layout.fillWidth: true; Layout.preferredHeight: 124
    radius: Style.cornerRadius; color: Color.background
    border.width: String(item.id) === root.service.selectedArtID ? 2 : 0
    border.color: Color.accent
    Image { anchors.fill: parent; anchors.margins: 3; source: "file://" + String(parent.item.image); fillMode: Image.PreserveAspectCrop; asynchronous: true; cache: true }
    Rectangle {
      z: 1
      visible: String(parent.item.id) === root.service.selectedArtID
      anchors.right: parent.right; anchors.top: parent.top; anchors.margins: 7
      width: 22; height: 22; radius: 11; color: Color.accent
      Text { anchors.centerIn: parent; text: "✓"; color: Color.background; font.bold: true }
    }
    PanelActionButton {
      z: 2
      visible: parent.deletable; anchors.left: parent.left; anchors.bottom: parent.bottom; anchors.margins: 7
      size: 28; iconText: "󰆴"; tooltipText: "Delete photo"; foreground: Color.urgent; hoverColor: Color.urgent; bordered: true
      onClicked: parent.deleteRequested(String(parent.item.id))
    }
    MouseArea { anchors.fill: parent; z: 0; cursorShape: Qt.PointingHandCursor; onClicked: parent.selected(String(parent.item.id)) }
  }

  Process {
    id: photoPicker
    command: ["/usr/bin/zenity", "--file-selection", "--title=Upload a photo to The Frame", "--file-filter=Images | *.jpg *.jpeg *.png"]
    stdout: StdioCollector { id: photoPickerOut; waitForEnd: true }
    onExited: function(code) { var path = String(photoPickerOut.text || "").trim(); if (code === 0 && path !== "" && root.service) root.service.uploadArt(path) }
  }
  onPopupOpenChanged: {
    if (!popupOpen) { remotePage = "navigate"; pendingDeleteID = ""; setupCapabilitiesOpen = false }
    if (service) {
      service.pollIntervalMs = Math.max(5, Math.min(120, Number(root.setting("pollSeconds", 15)))) * 1000
      service.panelOpen = popupOpen
      if (popupOpen) service.refresh()
    }
  }
  implicitWidth: barSize; implicitHeight: barSize
  opacity: service && service.snapshot.online ? 1 : 0.6

  Text { anchors.centerIn: parent; text: "󰐾"; color: root.bar ? root.bar.barForeground : Color.foreground; font.family: root.bar ? root.bar.fontFamily : Style.font.family; font.pixelSize: Style.font.body }
  WidgetButton {
    id: button; anchors.fill: parent; bar: root.bar; text: " "; labelVisible: false
    tooltipText: root.service && root.service.snapshot.online ? "Samsung Frame · online" : "Samsung Frame · offline"
    onPressed: function(mouseButton) { if (mouseButton === Qt.MiddleButton && root.service) root.service.refresh(); else root.toggle() }
  }

  KeyboardPanel {
    id: popup; anchorItem: button; bar: root.bar; owner: root; open: root.popupOpen; focusTarget: keyCatcher
    contentWidth: popup.fittedContentWidth(Math.max(400, Math.min(560, Number(root.setting("panelWidth", 430)))))
    contentHeight: popup.fittedContentHeight(content.implicitHeight, Math.max(520, Math.min(760, Number(root.setting("panelHeight", 680)))))
    PanelKeyCatcher { id: keyCatcher; anchors.fill: parent; onCloseRequested: root.close() }

    ColumnLayout {
      id: content; anchors.fill: parent; spacing: Style.space(10)
      RowLayout {
        Layout.fillWidth: true; spacing: Style.space(8)
        Rectangle { width: 8; height: 8; radius: 4; color: root.service && root.service.snapshot.online ? Color.accent : root.panelDim }
        ColumnLayout {
          Layout.fillWidth: true; spacing: 1
          Text { text: root.service && root.service.snapshot.device && root.service.snapshot.device.name ? root.service.snapshot.device.name : "Samsung The Frame"; color: root.panelForeground; font.family: root.uiFont; font.bold: true; font.pixelSize: Style.font.subtitle }
          Text { text: root.service ? root.service.message : "Connecting…"; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.caption }
        }
        PanelActionButton { iconText: "↻"; tooltipText: "Refresh"; focusable: true; foreground: root.panelForeground; onClicked: root.service && root.service.refresh() }
        PanelActionButton {
          iconText: root.service && root.service.snapshot.online ? "󰐥" : "󰑐"
          tooltipText: root.service && root.service.snapshot.online ? "Power off" : "Wake TV"
          focusable: true; foreground: root.panelForeground
          onClicked: root.service && (root.service.snapshot.online ? root.service.key("KEY_POWER") : root.service.wake())
        }
      }
      Text { visible: root.service && root.service.error !== ""; text: root.service ? root.service.error : ""; color: Color.urgent; wrapMode: Text.Wrap; Layout.fillWidth: true; font.family: root.uiFont; font.pixelSize: Style.font.caption }
      ButtonGroup {
        Layout.alignment: Qt.AlignHCenter; options: ["Remote", "Art", "Photos", "Setup"]
        value: root.page.charAt(0).toUpperCase() + root.page.slice(1)
        foreground: root.panelForeground; background: "transparent"; accent: Color.accent; fontFamily: root.uiFont; fontSize: Style.font.caption
        onChanged: function(value) { root.page = value.toLowerCase(); if ((root.page === "art" || root.page === "photos") && root.service && !root.service.galleryLoaded) root.service.loadGallery() }
      }

      ScrollView {
        id: scrollArea; Layout.fillWidth: true; Layout.fillHeight: true; clip: true
        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
        ScrollBar.vertical.policy: body.implicitHeight > height ? ScrollBar.AsNeeded : ScrollBar.AlwaysOff
        Binding { target: scrollArea.contentItem; property: "interactive"; value: body.implicitHeight > scrollArea.height }
        Column {
          id: body; width: scrollArea.availableWidth; spacing: Style.space(12)

          Column {
            visible: root.page === "remote"; width: parent.width; spacing: Style.space(16)
            ButtonGroup {
              anchors.horizontalCenter: parent.horizontalCenter; options: ["Navigate", "Sound", "Media", "TV"]
              value: root.remotePage.charAt(0).toUpperCase() + root.remotePage.slice(1)
              foreground: root.panelForeground; background: "transparent"; accent: Color.accent; fontFamily: root.uiFont; fontSize: Style.font.caption
              onChanged: function(value) { root.remotePage = value.toLowerCase() }
            }
            Column {
              visible: root.remotePage === "navigate"; width: parent.width; spacing: Style.space(14)
              GridLayout {
                columns: 3; anchors.horizontalCenter: parent.horizontalCenter; rowSpacing: 8; columnSpacing: 8
                Item { Layout.preferredWidth: 52; Layout.preferredHeight: 52 }
                NavButton { iconText: "▲"; onClicked: root.service.key("KEY_UP") }
                Item { Layout.preferredWidth: 52; Layout.preferredHeight: 52 }
                NavButton { iconText: "◀"; onClicked: root.service.key("KEY_LEFT") }
                NavButton { iconText: "OK"; onClicked: root.service.key("KEY_ENTER") }
                NavButton { iconText: "▶"; onClicked: root.service.key("KEY_RIGHT") }
                NavButton { iconText: "↩"; tooltipText: "Back"; onClicked: root.service.key("KEY_RETURN") }
                NavButton { iconText: "▼"; onClicked: root.service.key("KEY_DOWN") }
                NavButton { iconText: "⌂"; tooltipText: "Home"; onClicked: root.service.key("KEY_HOME") }
              }
              RowLayout {
                width: parent.width; spacing: Style.space(8)
                ActionButton { text: "Source"; Layout.fillWidth: true; onClicked: root.service.key("KEY_SOURCE") }
                ActionButton { text: "Rotate"; Layout.fillWidth: true; onClicked: root.service.rotate() }
              }
            }
            Column {
              visible: root.remotePage === "sound"; width: parent.width; spacing: Style.space(14)
              RowLayout {
                width: parent.width; spacing: Style.space(8)
                ActionButton { text: "Volume −"; Layout.fillWidth: true; onClicked: root.service.key("KEY_VOLDOWN") }
                ActionButton { text: "Mute"; Layout.fillWidth: true; onClicked: root.service.key("KEY_MUTE") }
                ActionButton { text: "Volume +"; Layout.fillWidth: true; onClicked: root.service.key("KEY_VOLUP") }
              }
              RowLayout {
                width: parent.width; spacing: Style.space(8)
                ActionButton { text: "Channel −"; Layout.fillWidth: true; onClicked: root.service.key("KEY_CHDOWN") }
                ActionButton { text: "Guide"; Layout.fillWidth: true; onClicked: root.service.key("KEY_GUIDE") }
                ActionButton { text: "Channel +"; Layout.fillWidth: true; onClicked: root.service.key("KEY_CHUP") }
              }
            }
            GridLayout {
              visible: root.remotePage === "media"; width: parent.width; columns: 3; rowSpacing: Style.space(8); columnSpacing: Style.space(8)
              ActionButton { text: "Previous"; Layout.fillWidth: true; onClicked: root.service.key("KEY_PREVIOUS") }
              ActionButton { text: "Play / Pause"; Layout.fillWidth: true; onClicked: root.service.key("KEY_PLAYPAUSE") }
              ActionButton { text: "Stop"; Layout.fillWidth: true; onClicked: root.service.key("KEY_STOP") }
              ActionButton { text: "Rewind"; Layout.fillWidth: true; onClicked: root.service.key("KEY_REWIND") }
              ActionButton { text: "Fast-forward"; Layout.fillWidth: true; onClicked: root.service.key("KEY_FF") }
            }
            GridLayout {
              visible: root.remotePage === "tv"; width: parent.width; columns: 2; rowSpacing: Style.space(8); columnSpacing: Style.space(8)
              Repeater {
                model: [{t:"Channel list",k:"KEY_CH_LIST"},{t:"Info",k:"KEY_INFO"},{t:"Menu",k:"KEY_MENU"},{t:"Tools",k:"KEY_TOOLS"},{t:"Previous channel",k:"KEY_PRECH"},{t:"Record",k:"KEY_REC"},{t:"Captions",k:"KEY_CAPTION"},{t:"Exit",k:"KEY_EXIT"}]
                ActionButton { required property var modelData; Layout.fillWidth: true; text: modelData.t; onClicked: root.service.key(modelData.k) }
              }
            }
          }

          Column {
            visible: root.page === "art"; width: parent.width; spacing: Style.space(10)
            onVisibleChanged: if (visible && root.service && !root.service.galleryLoaded) root.service.loadGallery()
            RowLayout {
              width: parent.width; spacing: Style.space(8)
              ActionButton { text: "Enter Art Mode"; Layout.fillWidth: true; onClicked: root.service.key("KEY_AMBIENT") }
              ActionButton { text: "Refresh"; Layout.fillWidth: true; onClicked: root.service.loadGallery() }
            }
            Text { visible: root.service && !root.service.galleryLoaded; width: parent.width; text: "Loading artwork…"; horizontalAlignment: Text.AlignHCenter; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.body }
            Text { visible: root.service && root.service.galleryLoaded && root.service.artGallery.length === 0; width: parent.width; text: "No artwork previews available."; horizontalAlignment: Text.AlignHCenter; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.body }
            GridLayout {
              visible: root.service && root.service.galleryLoaded; columns: 2; width: parent.width; rowSpacing: Style.space(8); columnSpacing: Style.space(8)
              Repeater { model: root.service ? root.service.artGallery : []; GalleryCard { required property var modelData; item: modelData; onSelected: id => root.service.selectArt(id) } }
            }
          }

          Column {
            visible: root.page === "photos"; width: parent.width; spacing: Style.space(10)
            onVisibleChanged: if (visible && root.service && !root.service.galleryLoaded) root.service.loadGallery()
            ActionButton { text: photoPicker.running ? "Opening picker…" : "Upload photo"; width: parent.width; enabled: !photoPicker.running; onClicked: photoPicker.running = true }
            RowLayout {
              width: parent.width; spacing: Style.space(8)
              ActionButton { text: "Shuffle · 5 min"; Layout.fillWidth: true; onClicked: root.service.slideshow(5, true) }
              ActionButton { text: "Every 15 min"; Layout.fillWidth: true; onClicked: root.service.slideshow(15, false) }
              ActionButton { text: "Stop"; Layout.fillWidth: true; onClicked: root.service.slideshow(0, false) }
            }
            RowLayout {
              visible: root.pendingDeleteID !== ""; width: parent.width; spacing: Style.space(8)
              Text { text: "Delete this photo?"; color: Color.urgent; font.family: root.uiFont; Layout.fillWidth: true }
              Button { text: "Cancel"; fontFamily: root.uiFont; onClicked: root.pendingDeleteID = "" }
              Button { text: "Delete"; fontFamily: root.uiFont; foreground: Color.urgent; onClicked: { var id = root.pendingDeleteID; root.pendingDeleteID = ""; root.service.deleteArt(id) } }
            }
            Text { visible: root.service && !root.service.galleryLoaded; width: parent.width; text: "Loading photos…"; horizontalAlignment: Text.AlignHCenter; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.body }
            Text { visible: root.service && root.service.galleryLoaded && root.service.photosGallery.length === 0; width: parent.width; text: "No personal photos yet."; horizontalAlignment: Text.AlignHCenter; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.body }
            GridLayout {
              visible: root.service && root.service.galleryLoaded; columns: 2; width: parent.width; rowSpacing: Style.space(8); columnSpacing: Style.space(8)
              Repeater {
                model: root.service ? root.service.photosGallery : []
                GalleryCard { required property var modelData; item: modelData; deletable: true; onSelected: id => root.service.selectArt(id); onDeleteRequested: id => root.pendingDeleteID = id }
              }
            }
          }

          Column {
            visible: root.page === "setup"; width: parent.width; spacing: Style.space(10)
            ActionButton { text: "Discover TVs"; width: parent.width; onClicked: root.service.discover() }
            Repeater { model: root.service ? root.service.devices : []; ActionButton { required property var modelData; width: parent.width; text: String(modelData.name || modelData.model || modelData.ip); onClicked: root.service.configure(String(modelData.ip)) } }
            TextField { id: ipField; width: parent.width; placeholderText: "TV IP address"; inputMethodHints: Qt.ImhFormattedNumbersOnly; font.family: root.uiFont }
            ActionButton { text: "Save address"; width: parent.width; enabled: ipField.text.trim() !== ""; onClicked: root.service.configure(ipField.text) }
            Text { width: parent.width; text: "Stand pairing requires a compatible physical Samsung Smart Remote."; wrapMode: Text.Wrap; color: root.panelDim; font.family: root.uiFont; font.pixelSize: Style.font.caption }
            Button { text: (root.setupCapabilitiesOpen ? "▾  " : "▸  ") + "Technical capabilities"; width: parent.width; leftAlign: true; fontFamily: root.uiFont; onClicked: root.setupCapabilitiesOpen = !root.setupCapabilitiesOpen }
            Column {
              visible: root.setupCapabilitiesOpen; width: parent.width; spacing: Style.space(8)
              Repeater {
                model: root.service ? root.service.capabilities : []
                Column {
                  required property var modelData; width: parent.width; spacing: 2
                  Text { text: String(parent.modelData.group || ""); font.bold: true; font.family: root.uiFont; color: root.panelForeground }
                  Text { text: String(parent.modelData.items || ""); width: parent.width; wrapMode: Text.Wrap; font.family: root.uiFont; color: root.panelDim }
                }
              }
            }
          }
        }
      }
    }
  }
}
