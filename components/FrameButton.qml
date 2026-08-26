import qs.Commons
import qs.Ui

Button {
    id: control

    property bool quiet: false

    implicitHeight: quiet ? Style.space(40) : Style.space(42)
    background: quiet ? "transparent" : Style.normalFillFor(foreground, accent)
    fontSize: quiet ? Style.font.bodySmall : Style.font.body
    bordered: false
    focusable: true
}
