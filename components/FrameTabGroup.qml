import QtQuick
import qs.Commons
import qs.Ui

Item {
    id: root

    property var options: []
    property string value: ""
    property color foreground: Color.foreground
    property color background: "transparent"
    property color accent: Color.accent
    property string fontFamily: Style.font.family
    property real fontSize: Style.font.body
    property int cursorIndex: -1

    signal changed(string value)

    activeFocusOnTab: true
    implicitWidth: group.implicitWidth
    implicitHeight: group.implicitHeight

    function optionValue(option) {
        return option && typeof option === "object" ? String(option.value) : String(option)
    }

    function selectedIndex() {
        for (var i = 0; i < options.length; i++)
            if (optionValue(options[i]) === value)
                return i;
        return -1;
    }

    function resetCursor() {
        var selected = selectedIndex();
        cursorIndex = selected < 0 && options.length > 0 ? 0 : selected;
    }

    function moveFocused(direction) {
        if (options.length === 0)
            return;
        if (cursorIndex < 0)
            resetCursor();
        cursorIndex = Math.max(0, Math.min(options.length - 1, cursorIndex + (direction < 0 ? -1 : 1)));
    }

    function activate() {
        if (cursorIndex < 0)
            resetCursor();
        if (cursorIndex >= 0 && cursorIndex < options.length)
            changed(optionValue(options[cursorIndex]));
    }

    onActiveFocusChanged: {
        if (activeFocus)
            resetCursor();
        else
            cursorIndex = -1;
    }

    ButtonGroup {
        id: group

        options: root.options
        value: root.value
        foreground: root.foreground
        background: root.background
        accent: root.accent
        fontFamily: root.fontFamily
        fontSize: root.fontSize
        focusable: false
        cursorIndex: root.activeFocus ? root.cursorIndex : -1
        onChanged: value => root.changed(value)
    }
}
