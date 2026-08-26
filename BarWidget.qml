import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Dialogs
import Quickshell
import qs.Commons
import qs.Ui

BarWidget {
  id: root
  moduleName: "swa.frame"
  readonly property var service: bar && bar.shell ? bar.shell.serviceFor("swa.frame") : null
  readonly property color panelForeground: bar ? bar.foreground : Color.foreground
  readonly property color panelDim: Qt.darker(panelForeground, 1.5)
  property bool popupOpen: false
  property string page: "remote"
  function close(){popupOpen=false} function open(){popupOpen=true} function toggle(){popupOpen=!popupOpen}
  readonly property bool opened: popupOpen
  FileDialog{id:photoPicker;title:"Upload a photo to The Frame";nameFilters:["Images (*.jpg *.jpeg *.png)"];onAccepted:if(service)service.uploadArt(selectedFile)}
  onPopupOpenChanged:{if(service){service.pollIntervalMs=Math.max(5,Math.min(120,Number(root.setting("pollSeconds",15))))*1000;service.panelOpen=popupOpen;if(popupOpen)service.refresh()}}
  implicitWidth: barSize; implicitHeight: barSize; opacity:service&&service.snapshot.online?1:0.6

  Text{anchors.centerIn:parent;text:"󰐾";color:root.bar?root.bar.barForeground:Color.foreground;font.family:root.bar?root.bar.fontFamily:Style.font.family;font.pixelSize:Style.font.body}
  WidgetButton{id:button;anchors.fill:parent;bar:root.bar;text:" ";labelVisible:false;tooltipText:service?(service.snapshot.online?"Samsung Frame · online":"Samsung Frame · offline"):"Samsung Frame";onPressed:function(mouseButton){if(mouseButton===Qt.MiddleButton&&service)service.refresh();else root.toggle()}}

  KeyboardPanel {
    id:popup;anchorItem:button;bar:root.bar;owner:root;open:root.popupOpen;focusTarget:keyCatcher
    contentWidth:popup.fittedContentWidth(Math.max(380,Math.min(620,Number(root.setting("panelWidth",430)))))
    contentHeight:popup.fittedContentHeight(content.implicitHeight,Math.max(520,Math.min(860,Number(root.setting("panelHeight",680)))))
    PanelKeyCatcher{id:keyCatcher;anchors.fill:parent;onCloseRequested:root.close()}
    ColumnLayout{id:content;anchors.fill:parent;spacing:Style.space(10)
      RowLayout{Layout.fillWidth:true
        ColumnLayout{Layout.fillWidth:true;spacing:2
          Text{text:service&&service.snapshot.device&&service.snapshot.device.name?service.snapshot.device.name:"Samsung The Frame";color:root.panelForeground;font.bold:true;font.pixelSize:Style.font.title}
          Text{text:service?service.message:"Starting…";color:service&&service.snapshot.online?Color.accent:root.panelDim;font.pixelSize:Style.font.caption}
        }
        Button{text:service&&service.snapshot.online?"Power off":"Wake";onClicked:service&&(service.snapshot.online?service.key("KEY_POWER"):service.wake())}
        Button{text:"↻";onClicked:service&&service.refresh()}
      }
      Text{visible:service&&service.error!=="";text:service?service.error:"";color:Color.urgent;wrapMode:Text.Wrap;Layout.fillWidth:true;font.pixelSize:Style.font.caption}
      ButtonGroup{Layout.alignment:Qt.AlignHCenter;spacing:Style.space(4);options:[{value:"remote",label:"Remote",icon:"󰒓"},{value:"art",label:"Art",icon:"󰏘"},{value:"setup",label:"Setup",icon:"󰒓"},{value:"api",label:"API",icon:"󰘦"}];value:root.page;foreground:root.panelForeground;background:"transparent";accent:Color.accent;fontFamily:root.bar.fontFamily;fontSize:Style.font.caption;onChanged:function(value){root.page=value;if(value==="art"&&service&&!service.galleryLoaded)service.loadGallery()}}
      ScrollView{id:scrollArea;Layout.fillWidth:true;Layout.fillHeight:true;clip:true;ScrollBar.horizontal.policy:ScrollBar.AlwaysOff;ScrollBar.vertical.policy:body.implicitHeight>height?ScrollBar.AsNeeded:ScrollBar.AlwaysOff
        Binding{target:scrollArea.contentItem;property:"interactive";value:body.implicitHeight>scrollArea.height}
        Column{id:body;width:scrollArea.availableWidth;spacing:Style.space(12)
          Column{visible:root.page==="remote";width:parent.width;spacing:Style.space(10)
            Text{text:"NAVIGATION";color:root.panelDim;font.pixelSize:Style.font.caption;font.bold:true;font.letterSpacing:1}
            GridLayout{columns:3;anchors.horizontalCenter:parent.horizontalCenter;rowSpacing:6;columnSpacing:6
              Item{Layout.preferredWidth:70;Layout.preferredHeight:38} Button{text:"▲";onClicked:service.key("KEY_UP")} Item{Layout.preferredWidth:70;Layout.preferredHeight:38}
              Button{text:"◀";onClicked:service.key("KEY_LEFT")} Button{text:"OK";onClicked:service.key("KEY_ENTER")} Button{text:"▶";onClicked:service.key("KEY_RIGHT")}
              Button{text:"Back";onClicked:service.key("KEY_RETURN")} Button{text:"▼";onClicked:service.key("KEY_DOWN")} Button{text:"Home";onClicked:service.key("KEY_HOME")}
            }
            Text{text:"CONTROLS";color:root.panelDim;font.pixelSize:Style.font.caption;font.bold:true;font.letterSpacing:1}
            GridLayout{columns:4;width:parent.width;rowSpacing:6;columnSpacing:6
              Repeater{model:[{"t":"Vol +","k":"KEY_VOLUP"},{"t":"Mute","k":"KEY_MUTE"},{"t":"Ch +","k":"KEY_CHUP"},{"t":"Guide","k":"KEY_GUIDE"},{"t":"Vol −","k":"KEY_VOLDOWN"},{"t":"Source","k":"KEY_SOURCE"},{"t":"Ch −","k":"KEY_CHDOWN"},{"t":"List","k":"KEY_CH_LIST"},{"t":"⏮","k":"KEY_PRECH"},{"t":"⏪","k":"KEY_REWIND"},{"t":"Play/Pause","k":"KEY_PLAYPAUSE"},{"t":"⏩","k":"KEY_FF"},{"t":"Stop","k":"KEY_STOP"},{"t":"Info","k":"KEY_INFO"},{"t":"Tools","k":"KEY_TOOLS"},{"t":"Menu","k":"KEY_MENU"}];Button{required property var modelData;text:modelData.t;Layout.fillWidth:true;onClicked:service.key(modelData.k)}}
            }
            Button{text:"Rotate portrait / landscape";width:parent.width;onClicked:service.rotate()}
            Text{text:"Rotation holds Multi View for 3 seconds. The Samsung stand must already be paired to the TV.";width:parent.width;wrapMode:Text.Wrap;color:root.panelDim;font.pixelSize:Style.font.caption}
          }
          Column{visible:root.page==="art";width:parent.width;spacing:Style.space(9);onVisibleChanged:if(visible&&service&&!service.galleryLoaded)service.loadGallery()
            RowLayout{width:parent.width
              Button{text:"Enter Art Mode";Layout.fillWidth:true;onClicked:service.key("KEY_AMBIENT")}
              Button{text:"Refresh gallery";Layout.fillWidth:true;onClicked:service.loadGallery()}
            }
            Button{text:"＋ Upload your photo";width:parent.width;onClicked:photoPicker.open()}
            Text{text:"MY PHOTOS SLIDESHOW";color:root.panelDim;font.pixelSize:Style.font.caption;font.bold:true;font.letterSpacing:1}
            RowLayout{width:parent.width
              Button{text:"5 min shuffle";Layout.fillWidth:true;onClicked:service.slideshow(5,true)}
              Button{text:"15 min";Layout.fillWidth:true;onClicked:service.slideshow(15,false)}
              Button{text:"Stop";Layout.fillWidth:true;onClicked:service.slideshow(0,false)}
            }
            Text{visible:service&&!service.galleryLoaded;width:parent.width;text:"Loading artwork from your TV…";horizontalAlignment:Text.AlignHCenter;color:root.panelDim;font.pixelSize:Style.font.body}
            GridLayout{visible:service&&service.galleryLoaded;columns:2;width:parent.width;rowSpacing:Style.space(8);columnSpacing:Style.space(8)
              Repeater{model:service?service.gallery:[]
                Rectangle{required property var modelData;required property int index;Layout.fillWidth:true;Layout.preferredHeight:120;color:Color.background;radius:Style.cornerRadius;border.width:String(modelData.id)===service.selectedArtID?2:1;border.color:String(modelData.id)===service.selectedArtID?Color.accent:root.panelDim
                  Image{anchors.fill:parent;anchors.margins:4;source:modelData.image?"file://"+String(modelData.image):"";fillMode:Image.PreserveAspectCrop;asynchronous:true;cache:true}
                  Text{visible:!modelData.image;anchors.centerIn:parent;text:"󰏘\nArtwork "+(index+1);horizontalAlignment:Text.AlignHCenter;color:root.panelDim;font.family:root.bar.fontFamily}
                  Rectangle{visible:String(modelData.id)===service.selectedArtID;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:6;width:24;height:24;radius:12;color:Color.accent
                    Text{anchors.centerIn:parent;text:"✓";color:Color.background;font.bold:true}
                  }
                  MouseArea{anchors.fill:parent;cursorShape:Qt.PointingHandCursor;onClicked:service.selectArt(modelData.id)}
                }
              }
            }
            Text{visible:service&&service.galleryLoaded;width:parent.width;text:"Uploads are stored by Samsung under My Photos. The slideshow includes every photo there; named app-only collections come after upload is confirmed on this TV.";wrapMode:Text.Wrap;color:root.panelDim;font.pixelSize:Style.font.caption}
          }
          Column{visible:root.page==="setup";width:parent.width;spacing:Style.space(9)
            Button{text:"Discover Frame TVs";width:parent.width;onClicked:service.discover()}
            Repeater{model:service?service.devices:[];Button{required property var modelData;width:parent.width;text:String(modelData.name||modelData.model||modelData.ip);onClicked:service.configure(String(modelData.ip))}}
            TextField{id:ipField;width:parent.width;placeholderText:"TV IP address, e.g. 192.168.1.50";inputMethodHints:Qt.ImhFormattedNumbersOnly}
            Button{text:"Save IP";width:parent.width;enabled:ipField.text.trim()!=="";onClicked:service.configure(ipField.text)}
            Text{text:"Initial stand pairing requires a compatible physical Samsung Smart Remote: hold Settings/Number/Color + Multi View together for at least 3 seconds. Firmware 1720.7 rejects that chord from network remotes.";width:parent.width;wrapMode:Text.Wrap;color:root.panelDim}
          }
          Column{visible:root.page==="api";width:parent.width;spacing:Style.space(9)
            Repeater{model:service?service.capabilities:[];ColumnLayout{required property var modelData;Layout.fillWidth:true;Text{text:String(modelData.group||"");font.bold:true;color:root.panelForeground}Text{text:String(modelData.items||"");wrapMode:Text.Wrap;Layout.fillWidth:true;color:root.panelDim}}}
            Text{visible:!service||service.capabilities.length===0;text:"Connect to the TV to load its capability reference.";color:root.panelDim}
          }
        }
      }
    }
  }
}
