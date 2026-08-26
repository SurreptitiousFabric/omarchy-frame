pragma ComponentBehavior: Bound

import QtQuick

Item {
    id: root

    property color stroke: "white"
    property real iconSize: 24

    readonly property real lineWidth: Math.max(1, Math.round(iconSize * 0.075))

    implicitWidth: iconSize
    implicitHeight: iconSize

    Rectangle {
        x: (root.width - width) / 2
        y: root.height * 0.02
        width: root.lineWidth
        height: root.height * 0.36
        radius: width / 2
        color: root.stroke
        antialiasing: true
        transform: Rotation {
            origin.x: root.lineWidth / 2
            origin.y: root.height * 0.36
            angle: -35
        }
    }
    Rectangle {
        x: (root.width - width) / 2
        y: root.height * 0.02
        width: root.lineWidth
        height: root.height * 0.36
        radius: width / 2
        color: root.stroke
        antialiasing: true
        transform: Rotation {
            origin.x: root.lineWidth / 2
            origin.y: root.height * 0.36
            angle: 35
        }
    }

    Rectangle {
        id: cabinet

        x: root.width * 0.05
        y: root.height * 0.34
        width: root.width * 0.90
        height: root.height * 0.61
        radius: Math.max(2, root.width * 0.12)
        color: "transparent"
        border.color: root.stroke
        border.width: root.lineWidth
        antialiasing: true

        Rectangle {
            x: cabinet.width * 0.12
            y: cabinet.height * 0.15
            width: cabinet.width * 0.76
            height: cabinet.height * 0.48
            radius: Math.max(1, root.width * 0.035)
            color: "transparent"
            border.color: root.stroke
            border.width: root.lineWidth
        }

        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            y: cabinet.height * 0.75
            spacing: root.width * 0.09

            Repeater {
                model: 3

                Rectangle {
                    required property int index

                    width: Math.max(1, root.width * 0.075)
                    height: width
                    radius: width / 2
                    color: root.stroke
                }
            }
        }
    }
}
