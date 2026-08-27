import QtQuick
import QtTest
import "../components"

Item {
    id: root

    width: 96
    height: 48

    Rectangle {
        id: smallCanvas
        x: 8
        y: 8
        width: 24
        height: 24
        color: "black"

        FrameTvIcon {
            anchors.fill: parent
            iconSize: 24
            stroke: "white"
        }
    }

    Rectangle {
        id: largeCanvas
        x: 48
        y: 8
        width: 32
        height: 32
        color: "black"

        FrameTvIcon {
            anchors.fill: parent
            iconSize: 32
            stroke: "white"
        }
    }

    TestCase {
        name: "FrameTvIconRender"
        when: windowShown

        function paintedPixels(image, x, y, width, height) {
            var count = 0;
            for (var py = y; py < y + height; py++) {
                for (var px = x; px < x + width; px++) {
                    if (image.red(px, py) + image.green(px, py) + image.blue(px, py) > 192)
                        count++;
                }
            }
            return count;
        }

        function verifyIcon(icon, size) {
            var image = grabImage(icon);
            verify(image.width >= size && image.width <= size * 4,
                   "unexpected rendered width");
            var deviceSize = image.width;
            compare(image.height, deviceSize);

            var top = Math.floor(deviceSize * 0.34);
            var cabinetHeight = deviceSize - top;
            verify(paintedPixels(image, 0, 0, Math.floor(deviceSize / 2), top) > 0,
                   "left antenna did not render");
            verify(paintedPixels(image, Math.floor(deviceSize / 2), 0,
                                 deviceSize - Math.floor(deviceSize / 2), top) > 0,
                   "right antenna did not render");
            verify(paintedPixels(image, 0, top, Math.ceil(deviceSize * 0.2), cabinetHeight) > 0,
                   "left cabinet edge did not render");
            verify(paintedPixels(image, Math.floor(deviceSize * 0.8), top,
                                 deviceSize - Math.floor(deviceSize * 0.8), cabinetHeight) > 0,
                   "right cabinet edge did not render");
            var buttonY = Math.floor(deviceSize * 0.77);
            var buttonHeight = Math.ceil(deviceSize * 0.16);
            verify(paintedPixels(image, Math.floor(deviceSize * 0.27), buttonY,
                                 Math.ceil(deviceSize * 0.13), buttonHeight) > 0,
                   "left cabinet button did not render");
            verify(paintedPixels(image, Math.floor(deviceSize * 0.44), buttonY,
                                 Math.ceil(deviceSize * 0.12), buttonHeight) > 0,
                   "center cabinet button did not render");
            verify(paintedPixels(image, Math.floor(deviceSize * 0.60), buttonY,
                                 Math.ceil(deviceSize * 0.13), buttonHeight) > 0,
                   "right cabinet button did not render");
            compare(image.red(0, 0), 0);
            compare(image.green(0, 0), 0);
            compare(image.blue(0, 0), 0);
            verify(paintedPixels(image, 0, 0, deviceSize, deviceSize) >= Math.floor(deviceSize * 2.5),
                   "icon produced too few painted pixels");
        }

        function test_renderAtBarSizes() {
            verifyIcon(smallCanvas, 24);
            verifyIcon(largeCanvas, 32);
        }
    }
}
