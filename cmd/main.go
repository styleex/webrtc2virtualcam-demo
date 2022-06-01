package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"

	gst "webrtc-go/internal/gstreamer-sink"
	"webrtc-go/internal/signal"
)

// gstreamerReceiveMain is launched in a goroutine because the main thread is needed
// for Glib's main loop (Gstreamer uses Glib)
func gstreamerReceiveMain(offerString string) string {
	// Everything below is the pion-WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Set a handler for when a new remote track starts, this handler creates a gstreamer pipeline
	// for the given codec
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		codecName := strings.Split(track.Codec().RTPCodecCapability.MimeType, "/")[1]
		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), codecName)
		pipeline := gst.CreatePipeline(track.PayloadType(), strings.ToLower(codecName))
		pipeline.Start()
		buf := make([]byte, 1400)
		for {
			i, _, readErr := track.Read(buf)
			if readErr != nil {
				panic(err)
			}

			pipeline.Push(buf[:i])
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	signal.Decode(offerString, &offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	return signal.Encode(*peerConnection.LocalDescription())
}

func init() {
	// This example uses Gstreamer's autovideosink element to display the received video
	// This element, along with some others, sometimes require that the process' main thread is used
	runtime.LockOSThread()
}

type M map[string]interface{}

func writeJson(w http.ResponseWriter, r *http.Request, data interface{}) {
	respBytes, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to serialize response: %s", err)
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	if _, err := w.Write(respBytes); err != nil {
		log.Printf("Failed to write resp to client: %s", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, respErr error) {
	respBytes, err := json.Marshal(M{
		"success": false,
		"error":   respErr.Error(),
	})
	if err != nil {
		log.Fatalf("Failed to serialize error response: %s (err=%s)", err, respErr.Error())
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(500)
	if _, err := w.Write(respBytes); err != nil {
		log.Printf("Failed to write resp to client: %s", err)
	}
}

func app() {
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	mux.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}

		var req struct {
			Offer string `json:"offer"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request: %s", err)
			writeError(w, r, fmt.Errorf("failed to decode request: %s", err))
			return
		}

		writeJson(w, r, M{
			"answer": gstreamerReceiveMain(req.Offer),
		})
	})

	log.Println("Listen on :8888")
	if err := http.ListenAndServe("0.0.0.0:8888", mux); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Start a new thread to do the actual work for this application
	go app()
	// Use this goroutine (which has been runtime.LockOSThread'd to he the main thread) to run the Glib loop that Gstreamer requires
	gst.StartMainLoop()
}
