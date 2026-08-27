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

    function findVisual(item, name) {
        if (!item)
            return null;
        if (item.objectName === name)
            return item;
        var descendants = item.children || [];
        for (var i = 0; i < descendants.length; i++) {
            var match = findVisual(descendants[i], name);
            if (match)
                return match;
        }
        return null;
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
            var icon = root.findVisual(root.widget, "frameBarIcon");
            if (!icon || icon.visible !== true || icon.opacity <= 0 || icon.width <= 0 || icon.height <= 0) {
                root.fail("bar icon is absent or invisible");
                return;
            }
            var clickTarget = root.findVisual(root.widget, "frameBarClickTarget");
            if (!clickTarget || clickTarget.visible !== true || clickTarget.opacity <= 0 ||
                    clickTarget.width <= 0 || clickTarget.height <= 0) {
                root.fail("bar click target is absent or inactive");
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
