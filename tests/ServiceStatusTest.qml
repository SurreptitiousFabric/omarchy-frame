pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

ShellRoot {
    id: root

    property var service: null

    function fail(message) {
        console.log("FRAME_SERVICE_STATUS_TEST_FAIL " + message);
        Qt.exit(1);
    }

    function expectEqual(actual, expected, message) {
        if (actual !== expected)
            fail(message + ": got " + actual + ", expected " + expected);
    }

    function runTest() {
        var component = Qt.createComponent(Qt.resolvedUrl("Service.qml"), Component.PreferSynchronous);
        if (component.status !== Component.Ready) {
            fail("component: " + component.errorString());
            return;
        }
        service = component.createObject(host);
        if (!service) {
            fail("createObject returned null");
            return;
        }

        expectEqual(service.statusMessage({ok: true, online: false, mode: "offline"}), "TV is offline", "offline message");
        expectEqual(service.statusMessage({ok: true, online: true, mode: "art", power: "standby"}), "Connected locally", "Art message");
        expectEqual(service.statusMessage({ok: true, online: true, mode: "tv", power: "on"}), "Connected locally", "TV message");
        expectEqual(service.statusMessage({ok: true, online: true, mode: "unknown", power: "on"}), "Connected · TV on", "ambiguous powered-on message");
        expectEqual(service.statusMessage({ok: true, online: true, mode: "unknown", power: "standby"}), "Connected locally", "ambiguous standby message");

        console.log("FRAME_SERVICE_STATUS_TEST_PASS");
        service.destroy();
        service = null;
        Qt.quit();
    }

    Item {
        id: host
    }

    Timer {
        interval: 4000
        running: true
        repeat: false
        onTriggered: root.fail("timeout")
    }

    Component.onCompleted: Qt.callLater(runTest)
}
