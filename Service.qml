import QtQuick
import Quickshell
import Quickshell.Io

Item {
  id: root
  property var shell: null
  readonly property string pluginPath: Quickshell.env("HOME") + "/.config/omarchy/plugins/swa.frame"
  readonly property string backend: pluginPath + "/bin/frame-controller"
  property var snapshot: ({ok: false, online: false, device: {}})
  property var devices: []
  property var gallery: []
  property var artGallery: []
  property var photosGallery: []
  property bool galleryLoaded: false
  property string selectedArtID: ""
  property var capabilities: []
  property bool busy: statusProcess.running || actionProcess.running || discoverProcess.running
  property string message: "Searching for The Frame…"
  property string error: ""
  property bool panelOpen: false
  property var pending: []
  property int pollIntervalMs: 15000

  function parse(text) { try { return JSON.parse(String(text || "").trim()) } catch (_) { return null } }
  function compact(text) { var s=String(text||"").replace(/\s+/g," ").trim(); return s.length>180?s.substring(0,177)+"…":s }
  function run(args, success) {
    if (actionProcess.running) {
      if (pending.length >= 20) { error = "Too many queued commands"; return }
      pending.push({args:args, success:success}); return
    }
    actionProcess.command = [backend].concat(args); actionProcess.successText = success || "Done"; actionProcess.running = true
  }
  function next() { if (!pending.length) return; var q=pending.shift(); run(q.args,q.success) }
  function refresh() { if (!statusProcess.running) { statusProcess.command=[backend,"status"];statusProcess.running=true } }
  function discover() { if (!discoverProcess.running) { error="";message="Searching your network…";discoverProcess.command=[backend,"discover"];discoverProcess.running=true } }
  function configure(ip) { run(["configure",String(ip).trim()],"TV saved — approve the next command on screen") }
  function key(name) { run(["key",name],"") }
  function rotate() { run(["rotate"],"Rotation toggled") }
  function wake() { run(["wake"],"Wake packet sent") }
  function loadGallery() { run(["gallery"],"") }
  function setGallery(items) {
    gallery = items || []
    var arts = []
    var photos = []
    selectedArtID = ""
    for (var i = 0; i < gallery.length; i++) {
      var item = gallery[i]
      if (item.current) selectedArtID = String(item.id)
      if (!item.image) continue
      if (String(item.category) === "MY-C0002") photos.push(item)
      else arts.push(item)
    }
    artGallery = arts
    photosGallery = photos
    galleryLoaded = true
  }
  function selectArt(id) { selectedArtID=String(id);run(["select-art",selectedArtID],"Artwork selected") }
  function uploadArt(fileUrl) { run(["upload-art",String(fileUrl)],"Photo uploaded and selected") }
  function deleteArt(id) { run(["delete-art",String(id)],"Photo deleted from the TV") }
  function slideshow(minutes, shuffle) { run(["slideshow",String(minutes),shuffle?"shuffle":"sequential"],minutes>0?"My Photos slideshow started":"Slideshow stopped") }
  function art(request, value) { var a=["art",request];if(value!==undefined&&value!=="")a.push(String(value));run(a,"Art request sent") }

  Process {
    id: statusProcess
    command: []
    stdout: StdioCollector{id:statusOut;waitForEnd:true}
    stderr: StdioCollector{id:statusErr;waitForEnd:true}
    onExited: function(code){var p=root.parse(statusOut.text);if(code===0&&p){root.snapshot=p;root.capabilities=p.capabilities||[];root.error="";root.message=p.online?"Connected locally":"TV is offline"}else{root.error=root.compact((p&&p.error)||statusErr.text||"TV not configured");if(root.devices.length===0)Qt.callLater(root.discover)}}
  }
  Process {
    id: discoverProcess
    command: []
    stdout:StdioCollector{id:discoverOut;waitForEnd:true}
    stderr:StdioCollector{id:discoverErr;waitForEnd:true}
    onExited:function(code){var p=root.parse(discoverOut.text);root.devices=p&&p.devices?p.devices:[];if(root.devices.length===1){root.configure(root.devices[0].ip)}else if(root.devices.length>1){root.message="Choose a TV"}else{root.message="No Frame found — enter its IP";root.error=code===0?"":root.compact((p&&p.error)||discoverErr.text)}}
  }
  Process {
    id: actionProcess
    property string successText:""
    command:[]
    stdout:StdioCollector{id:actionOut;waitForEnd:true}
    stderr:StdioCollector{id:actionErr;waitForEnd:true}
    onExited:function(code){
      var p=root.parse(actionOut.text)
      if(code===0&&p&&p.ok){
        root.error=""
        if(p.items!==undefined)root.setGallery(p.items)
        if(p.content_id!==undefined||p.deleted_id!==undefined){root.galleryLoaded=false;root.pending.unshift({args:["gallery"],success:""})}
        if(p.response)root.message=root.compact(JSON.stringify(p.response));else if(successText!=="")root.message=successText;else if(p.message)root.message=p.message
      }else root.error=root.compact((p&&p.error)||actionErr.text||"Command failed")
      refreshTimer.restart();Qt.callLater(root.next)
    }
  }
  Timer{id:refreshTimer;interval:900;onTriggered:root.refresh()}
  Timer{interval:root.pollIntervalMs;repeat:true;running:true;onTriggered:if(root.panelOpen)root.refresh()}
  Component.onCompleted: refresh()
}
