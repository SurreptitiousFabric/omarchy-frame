pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import "components"

ShellRoot {
    id: root

    property string activatedValue: ""
    property bool finished: false

    FrameTabGroup {
        id: tabs

        options: [
            { label: "Remote", value: "remote" },
            { label: "Art", value: "art" },
            { label: "Photos", value: "photos" },
            { label: "Setup", value: "setup" }
        ]
        value: "remote"
        onChanged: value => root.activatedValue = value
    }

    function fail(message) {
        if (finished)
            return;
        finished = true;
        console.log("FRAME_TAB_TEST_FAIL " + message);
        Qt.quit();
    }

    function expect(condition, message) {
        if (!condition) {
            fail(message);
            return false;
        }
        return true;
    }

    function run() {
        tabs.resetCursor();
        if (!expect(tabs.cursorIndex === 0, "selected tab was not the initial cursor")) return;

        tabs.moveFocused(1);
        if (!expect(tabs.cursorIndex === 1, "right movement did not select Art")) return;
        tabs.activate();
        if (!expect(activatedValue === "art", "activation did not emit the focused value")) return;

        tabs.cursorIndex = 3;
        tabs.moveFocused(1);
        if (!expect(tabs.cursorIndex === 3, "right movement escaped the upper bound")) return;

        tabs.cursorIndex = 0;
        tabs.moveFocused(-1);
        if (!expect(tabs.cursorIndex === 0, "left movement escaped the lower bound")) return;

        tabs.value = "unknown";
        tabs.resetCursor();
        if (!expect(tabs.cursorIndex === 0, "unknown value did not fall back to the first tab")) return;

        finished = true;
        console.log("FRAME_TAB_TEST_PASS");
        Qt.quit();
    }

    Component.onCompleted: Qt.callLater(run)
}
