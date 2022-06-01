/* eslint-env browser */

let pc = new RTCPeerConnection({
  iceServers: [
  ]
})
let log = msg => {
  document.getElementById('logs').innerHTML += msg + '<br>'
}
let displayVideo = video => {
  var el = document.createElement('video')
  el.srcObject = video
  el.autoplay = true
  el.muted = true
  el.width = 160
  el.height = 120

  document.getElementById('localVideos').appendChild(el)
  return video
}

navigator.mediaDevices.getUserMedia({ video: true, audio: true })
  .then(stream => {

    stream.getTracks().forEach(function(track) {
      pc.addTrack(track, stream);
    });

    displayVideo(stream)
    pc.createOffer().then(d => pc.setLocalDescription(d)).catch(log)
  }).catch(log)

pc.oniceconnectionstatechange = e => log(pc.iceConnectionState)
pc.onicecandidate = event => {
  if (event.candidate === null) {
    document.getElementById('localSessionDescription').value = btoa(JSON.stringify(pc.localDescription))
  }
}

window.startSession = () => {
  const offer = document.getElementById('localSessionDescription').value;
  fetch("/call", {
    method: "POST",
    body: JSON.stringify({
      "offer": offer,
    })
  }).then(async (resp) => {
    const respData = await resp.json();
    try {
      await pc.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(respData.answer))))
    } catch (e) {
      alert(e)
    }
  })
}

window.addDisplayCapture = () => {
  navigator.mediaDevices.getDisplayMedia().then(stream => {
    document.getElementById('displayCapture').disabled = true

    stream.getTracks().forEach(function(track) {
     pc.addTrack(track, displayVideo(stream));
    });

    pc.createOffer().then(d => pc.setLocalDescription(d)).catch(log)
  })
}
