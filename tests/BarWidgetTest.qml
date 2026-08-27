pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

ShellRoot {
    id: root

    property var widget: null

    function fail(message) {
        console.log("FRAME_BAR_WIDGET_TEST_FAIL " + message);
        Qt.exit(1);
    }

    function loadWidget() {
        var component = Qt.createComponent(Qt.resolvedUrl("BarWidget.qml"), Component.PreferSynchronous);
        if (component.status !== Component.Ready) {
            fail("component: " + component.errorString());
            return;
        }

        widget = component.createObject(host, {
            "bar": mockBar,
            "settings": ({})
        });
        if (!widget) {
            fail("createObject returned null");
            return;
        }

        Qt.callLater(function() {
            if (!root.widget) {
                root.fail("widget disappeared");
                return;
            }
            if (root.widget.moduleName !== "io.github.surreptitiousfabric.omarchy-frame") {
                root.fail("unexpected module name");
                return;
            }
            if (root.widget.implicitWidth <= 0 || root.widget.implicitHeight <= 0) {
                root.fail("widget has no bar geometry");
                return;
            }

            console.log("FRAME_BAR_WIDGET_TEST_PASS");
            root.widget.destroy();
            root.widget = null;
            Qt.quit();
        });
    }

    QtObject {
        id: mockShell

        function serviceFor(pluginId) {
            return null;
        }
    }

    QtObject {
        id: mockBar

        property var shell: mockShell
        property int barSize: 32
        property color foreground: "#eeeeee"
        property color barForeground: "#eeeeee"
        property color urgent: "#ff5555"
        property string fontFamily: "sans-serif"
        property string position: "top"
        property bool vertical: false
        property bool foregroundAnimationEnabled: false
        property var activePopout: null

        function registerClickTarget(item) {}
        function unregisterClickTarget(item) {}
        function showTooltip(item, text) {}
        function hideTooltip(item) {}
        function requestPopout(item) {
            activePopout = item;
        }
        function releasePopout(item) {
            if (activePopout === item)
                activePopout = null;
        }
        function switchPanelFrom(item, direction) {
            return false;
        }
    }

    Item {
        id: host
        width: mockBar.barSize
        height: mockBar.barSize
    }

    Timer {
        interval: 4000
        running: true
        repeat: false
        onTriggered: root.fail("timeout")
    }

    Component.onCompleted: Qt.callLater(loadWidget)
}
